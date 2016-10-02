package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "zpool.h"
// #include "zfs.h"
import "C"

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"unsafe"
)

const (
	msgPoolIsNil = "Pool handle not initialized or its closed"
)

// PoolProperties type is map of pool properties name -> value
type PoolProperties map[Prop]string

/*
 * ZIO types.  Needed to interpret vdev statistics below.
 */
const (
	ZIOTypeNull = iota
	ZIOTypeRead
	ZIOTypeWrite
	ZIOTypeFree
	ZIOTypeClaim
	ZIOTypeIOCtl
	ZIOTypes
)

// Scan states
const (
	DSSNone      = iota // No scan
	DSSScanning         // Scanning
	DSSFinished         // Scan finished
	DSSCanceled         // Scan canceled
	DSSNumStates        // Total number of scan states
)

// Scan functions
const (
	PoolScanNone     = iota // No scan function
	PoolScanScrub           // Pools is checked against errors
	PoolScanResilver        // Pool is resilvering
	PoolScanFuncs           // Number of scan functions
)

// VDevStat - Vdev statistics.  Note: all fields should be 64-bit because this
// is passed between kernel and userland as an nvlist uint64 array.
type VDevStat struct {
	Timestamp      time.Duration    /* time since vdev load	(nanoseconds)*/
	State          VDevState        /* vdev state		*/
	Aux            VDevAux          /* see vdev_aux_t	*/
	Alloc          uint64           /* space allocated	*/
	Space          uint64           /* total capacity	*/
	DSpace         uint64           /* deflated capacity	*/
	RSize          uint64           /* replaceable dev size */
	ESize          uint64           /* expandable dev size */
	Ops            [ZIOTypes]uint64 /* operation count	*/
	Bytes          [ZIOTypes]uint64 /* bytes read/written	*/
	ReadErrors     uint64           /* read errors		*/
	WriteErrors    uint64           /* write errors		*/
	ChecksumErrors uint64           /* checksum errors	*/
	SelfHealed     uint64           /* self-healed bytes	*/
	ScanRemoving   uint64           /* removing?	*/
	ScanProcessed  uint64           /* scan processed bytes	*/
	Fragmentation  uint64           /* device fragmentation */
}

// PoolScanStat - Pool scan statistics.  Corresponds to `pool_scan_stat_t` in `include/sys/fs/zfs.h`.
type PoolScanStat struct {
	// Values stored on disk
	Func      PoolScanFunc // Current scan function e.g. none, scrub ...
	State     DSLScanState // Current scan state e.g. scanning, finished ...
	StartTime uint64       // Scan start time [@KK TODO: convert to time.Duration?]
	EndTime   uint64       // Scan end time [@KK TODO: convert to time.Duration?]
	ToExamine uint64       // Total bytes to scan
	Examined  uint64       // Total bytes scaned
	ToProcess uint64       // Total bytes to processed
	Processed uint64       // Total bytes processed
	Errors    uint64       // Scan errors
	// Values not stored on disk
	PassExam  uint64 // Examined bytes per scan pass
	PassStart uint64 // Start time of scan pass
}

// VDevTree ZFS virtual device tree
type VDevTree struct {
	Type     VDevType
	Devices  []VDevTree // groups other devices (e.g. mirror)
	Parity   uint
	Path     string
	Name     string
	Stat     VDevStat
	ScanStat PoolScanStat
}

// ExportedPool is type representing ZFS pool available for import
type ExportedPool struct {
	VDevs   VDevTree
	Name    string
	Comment string
	GUID    uint64
	State   PoolState
	Status  PoolStatus
}

// Pool object represents handler to single ZFS pool
//
/* Pool.Properties map[string]Property
 */
// Map of all ZFS pool properties, changing any of this will not affect ZFS
// pool, for that use SetProperty( name, value string) method of the pool
// object. This map is initial loaded when ever you open or create pool to
// give easy access to listing all available properties. It can be refreshed
// with up to date values with call to (*Pool) ReloadProperties
type Pool struct {
	list       *C.zpool_list_t
	Properties []Property
	Features   map[string]string
}

// PoolOpen open ZFS pool handler by name.
// Returns Pool object, requires Pool.Close() to be called explicitly
// for memory cleanup after object is not needed anymore.
func PoolOpen(name string) (pool Pool, err error) {
	csName := C.CString(name)
	defer C.free(unsafe.Pointer(csName))
	pool.list = C.zpool_list_open(libzfsHandle, csName)

	if pool.list != nil {
		err = pool.ReloadProperties()
		return
	}
	err = LastError()
	return
}

func poolGetConfig(name string, nv *C.nvlist_t) (vdevs VDevTree, err error) {
	var dtype *C.char
	var c, children C.uint_t
	var notpresent C.uint64_t
	var vs *C.vdev_stat_t
	var ps *C.pool_scan_stat_t
	var child **C.nvlist_t
	if 0 != C.nvlist_lookup_string(nv, C.sZPOOL_CONFIG_TYPE, &dtype) {
		err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_TYPE)
		return
	}
	vdevs.Name = name
	vdevs.Type = VDevType(C.GoString(dtype))
	if vdevs.Type == VDevTypeMissing || vdevs.Type == VDevTypeHole {
		return
	}

	// Fetch vdev state
	if 0 != C.nvlist_lookup_uint64_array_vds(nv, C.sZPOOL_CONFIG_VDEV_STATS,
		&vs, &c) {
		err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_VDEV_STATS)
		return
	}
	vdevs.Stat.Timestamp = time.Duration(vs.vs_timestamp)
	vdevs.Stat.State = VDevState(vs.vs_state)
	vdevs.Stat.Aux = VDevAux(vs.vs_aux)
	vdevs.Stat.Alloc = uint64(vs.vs_alloc)
	vdevs.Stat.Space = uint64(vs.vs_space)
	vdevs.Stat.DSpace = uint64(vs.vs_dspace)
	vdevs.Stat.RSize = uint64(vs.vs_rsize)
	vdevs.Stat.ESize = uint64(vs.vs_esize)
	for z := 0; z < ZIOTypes; z++ {
		vdevs.Stat.Ops[z] = uint64(vs.vs_ops[z])
		vdevs.Stat.Bytes[z] = uint64(vs.vs_bytes[z])
	}
	vdevs.Stat.ReadErrors = uint64(vs.vs_read_errors)
	vdevs.Stat.WriteErrors = uint64(vs.vs_write_errors)
	vdevs.Stat.ChecksumErrors = uint64(vs.vs_checksum_errors)
	vdevs.Stat.SelfHealed = uint64(vs.vs_self_healed)
	vdevs.Stat.ScanRemoving = uint64(vs.vs_scan_removing)
	vdevs.Stat.ScanProcessed = uint64(vs.vs_scan_processed)
	vdevs.Stat.Fragmentation = uint64(vs.vs_fragmentation)

	// Fetch vdev scan stats
	if 0 == C.nvlist_lookup_uint64_array_ps(nv, C.sZPOOL_CONFIG_SCAN_STATS,
		&ps, &c) {
		vdevs.ScanStat.Func = PoolScanFunc(ps.pss_func)
		vdevs.ScanStat.State = DSLScanState(ps.pss_state)
		vdevs.ScanStat.StartTime = uint64(ps.pss_start_time)
		vdevs.ScanStat.EndTime = uint64(ps.pss_end_time)
		vdevs.ScanStat.ToExamine = uint64(ps.pss_to_examine)
		vdevs.ScanStat.Examined = uint64(ps.pss_examined)
		vdevs.ScanStat.ToProcess = uint64(ps.pss_to_process)
		vdevs.ScanStat.Processed = uint64(ps.pss_processed)
		vdevs.ScanStat.Errors = uint64(ps.pss_errors)
		vdevs.ScanStat.PassExam = uint64(ps.pss_pass_exam)
		vdevs.ScanStat.PassStart = uint64(ps.pss_pass_start)
	}

	// Fetch the children
	if C.nvlist_lookup_nvlist_array(nv, C.sZPOOL_CONFIG_CHILDREN,
		&child, &children) != 0 {
		return
	}
	if children > 0 {
		vdevs.Devices = make([]VDevTree, 0, children)
	}
	if C.nvlist_lookup_uint64(nv, C.sZPOOL_CONFIG_NOT_PRESENT,
		&notpresent) == 0 {
		var path *C.char
		if 0 != C.nvlist_lookup_string(nv, C.sZPOOL_CONFIG_PATH, &path) {
			err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_PATH)
			return
		}
		vdevs.Path = C.GoString(path)
	}
	for c = 0; c < children; c++ {
		var islog = C.uint64_t(C.B_FALSE)

		C.nvlist_lookup_uint64(C.nvlist_array_at(child, c),
			C.sZPOOL_CONFIG_IS_LOG, &islog)
		if islog != C.B_FALSE {
			continue
		}
		vname := C.zpool_vdev_name(libzfsHandle, nil, C.nvlist_array_at(child, c),
			C.B_TRUE)
		var vdev VDevTree
		vdev, err = poolGetConfig(C.GoString(vname),
			C.nvlist_array_at(child, c))
		C.free(unsafe.Pointer(vname))
		if err != nil {
			return
		}
		vdevs.Devices = append(vdevs.Devices, vdev)
	}
	return
}

// PoolImportSearch - Search pools available to import but not imported.
// Returns array of found pools.
func PoolImportSearch(searchpaths []string) (epools []ExportedPool, err error) {
	var config, nvroot *C.nvlist_t
	var cname, msgid, comment *C.char
	var poolState, guid C.uint64_t
	var reason C.zpool_status_t
	var errata C.zpool_errata_t
	config = nil
	var elem *C.nvpair_t
	numofp := len(searchpaths)
	cpaths := C.alloc_cstrings(C.int(numofp))
	defer C.free(unsafe.Pointer(cpaths))
	for i, path := range searchpaths {
		csPath := C.CString(path)
		defer C.free(unsafe.Pointer(csPath))
		C.strings_setat(cpaths, C.int(i), csPath)
	}

	pools := C.zpool_find_import(libzfsHandle, C.int(numofp), cpaths)
	defer C.nvlist_free(pools)
	elem = C.nvlist_next_nvpair(pools, elem)
	epools = make([]ExportedPool, 0, 1)
	for ; elem != nil; elem = C.nvlist_next_nvpair(pools, elem) {
		ep := ExportedPool{}
		if C.nvpair_value_nvlist(elem, &config) != 0 {
			err = LastError()
			return
		}
		if C.nvlist_lookup_uint64(config, C.sZPOOL_CONFIG_POOL_STATE,
			&poolState) != 0 {
			err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_POOL_STATE)
			return
		}
		ep.State = PoolState(poolState)
		if C.nvlist_lookup_string(config, C.sZPOOL_CONFIG_POOL_NAME, &cname) != 0 {
			err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_POOL_NAME)
			return
		}
		ep.Name = C.GoString(cname)
		if C.nvlist_lookup_uint64(config, C.sZPOOL_CONFIG_POOL_GUID, &guid) != 0 {
			err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_POOL_GUID)
			return
		}
		ep.GUID = uint64(guid)
		reason = C.zpool_import_status(config, &msgid, &errata)
		ep.Status = PoolStatus(reason)

		if C.nvlist_lookup_string(config, C.sZPOOL_CONFIG_COMMENT, &comment) == 0 {
			ep.Comment = C.GoString(comment)
		}

		if C.nvlist_lookup_nvlist(config, C.sZPOOL_CONFIG_VDEV_TREE,
			&nvroot) != 0 {
			err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_VDEV_TREE)
			return
		}
		ep.VDevs, err = poolGetConfig(ep.Name, nvroot)
		epools = append(epools, ep)
	}
	return
}

func poolSearchImport(q string, searchpaths []string, guid bool) (name string,
	err error) {
	var config *C.nvlist_t
	var cname *C.char
	config = nil
	errPoolList := errors.New("Failed to list pools")
	var elem *C.nvpair_t
	numofp := len(searchpaths)
	cpaths := C.alloc_cstrings(C.int(numofp))
	defer C.free(unsafe.Pointer(cpaths))
	for i, path := range searchpaths {
		csPath := C.CString(path)
		defer C.free(unsafe.Pointer(csPath))
		C.strings_setat(cpaths, C.int(i), csPath)
	}

	pools := C.zpool_find_import(libzfsHandle, C.int(numofp), cpaths)
	defer C.nvlist_free(pools)

	elem = C.nvlist_next_nvpair(pools, elem)
	for ; elem != nil; elem = C.nvlist_next_nvpair(pools, elem) {
		var cq *C.char
		var tconfig *C.nvlist_t
		retcode := C.nvpair_value_nvlist(elem, &tconfig)
		if retcode != 0 {
			err = errPoolList
			return
		}
		if guid {
			var iguid C.uint64_t
			if retcode = C.nvlist_lookup_uint64(tconfig,
				C.sZPOOL_CONFIG_POOL_GUID, &iguid); retcode != 0 {
				err = errPoolList
				return
			}
			sguid := fmt.Sprint(iguid)
			if q == sguid {
				config = tconfig
				break
			}
		} else {
			if retcode = C.nvlist_lookup_string(tconfig,
				C.sZPOOL_CONFIG_POOL_NAME, &cq); retcode != 0 {
				err = errPoolList
				return
			}
			cname = cq
			name = C.GoString(cq)
			if q == name {
				config = tconfig
				break
			}
		}
	}
	if config == nil {
		err = fmt.Errorf("No pool found %s", q)
		return
	}
	if guid {
		// We need to get name so we can open pool by name
		if retcode := C.nvlist_lookup_string(config,
			C.sZPOOL_CONFIG_POOL_NAME, &cname); retcode != 0 {
			err = errPoolList
			return
		}
		name = C.GoString(cname)
	}
	if retcode := C.zpool_import(libzfsHandle, config, cname,
		nil); retcode != 0 {
		err = LastError()
		return
	}
	return
}

// PoolImport given a list of directories to search, find and import pool with matching
// name stored on disk.
func PoolImport(name string, searchpaths []string) (pool Pool, err error) {
	_, err = poolSearchImport(name, searchpaths, false)
	if err != nil {
		return
	}
	pool, err = PoolOpen(name)
	return
}

// PoolImportByGUID given a list of directories to search, find and import pool
// with matching GUID stored on disk.
func PoolImportByGUID(guid string, searchpaths []string) (pool Pool, err error) {
	var name string
	name, err = poolSearchImport(guid, searchpaths, true)
	if err != nil {
		return
	}
	pool, err = PoolOpen(name)
	return
}

// func PoolList(paths []string, cache string) (pools []Pool, err error) {
//
// }

// PoolOpenAll open all active ZFS pools on current system.
// Returns array of Pool handlers, each have to be closed after not needed
// anymore. Call Pool.Close() method.
func PoolOpenAll() (pools []Pool, err error) {
	var pool Pool
	if libzfsHandle == nil {
		return pools, fmt.Errorf("libzfs unitialized, missing privs?")
	}
	errcode := C.zpool_list(libzfsHandle, &pool.list)
	for pool.list != nil {
		err = pool.ReloadProperties()
		if err != nil {
			return
		}
		pools = append(pools, pool)
		pool.list = C.zpool_next(pool.list)
	}
	if errcode != 0 {
		err = LastError()
	}
	return
}

// PoolCloseAll close all pools in given slice
func PoolCloseAll(pools []Pool) {
	for _, p := range pools {
		p.Close()
	}
}

// PoolPropertyToName convert property to name
// ( returns built in string representation of property name).
// This is optional, you can represent each property with string
// name of choice.
func PoolPropertyToName(p Prop) (name string) {
	if p == PoolNumProps {
		return "numofprops"
	}
	prop := C.zpool_prop_t(p)
	name = C.GoString(C.zpool_prop_to_name(prop))
	return
}

// PoolStateToName maps POOL STATE to string.
func PoolStateToName(state PoolState) (name string) {
	ps := C.pool_state_t(state)
	name = C.GoString(C.zpool_pool_state_to_name(ps))
	return
}

// RefreshStats the pool's vdev statistics, e.g. bytes read/written.
func (pool *Pool) RefreshStats() (err error) {
	if 0 != C.refresh_stats(pool.list) {
		return errors.New("error refreshing stats")
	}
	return nil
}

// ReloadProperties re-read ZFS pool properties and features, refresh
// Pool.Properties and Pool.Features map
func (pool *Pool) ReloadProperties() (err error) {
	zph := pool.list.zph
	propList := C.read_zpool_properties(zph)
	// log.Printf("YYY reloadprop 0a")
	if propList == nil {
		// log.Printf("YYY reloadprop 0b")
		err = LastError()
		return
	}

	pool.Properties = make([]Property, PoolNumProps+1)
	next := propList
	for next != nil {
		pool.Properties[next.property] = Property{Value: C.GoString(&(next.value[0])), Source: C.GoString(&(next.source[0]))}
		next = C.next_property(next)
	}
	C.free_properties(propList)

	// read features
	pool.Features = map[string]string{
		"async_destroy":      "disabled",
		"empty_bpobj":        "disabled",
		"lz4_compress":       "disabled",
		"spacemap_histogram": "disabled",
		"enabled_txg":        "disabled",
		"hole_birth":         "disabled",
		"extensible_dataset": "disabled",
		"embedded_data":      "disabled",
		"bookmarks":          "disabled",
		"filesystem_limits":  "disabled",
		"large_blocks":       "disabled"}
	for name := range pool.Features {
		_, ferr := pool.GetFeature(name)
		if ferr != nil {
			// tolerate it
		}
	}

	return
}

// GetProperty reload and return single specified property. This also reloads requested
// property in Properties map.
func (pool *Pool) GetProperty(p Prop) (prop Property, err error) {
	if pool.list != nil {
		// First check if property exist at all
		if p < PoolPropName || p > PoolNumProps {
			err = errors.New(fmt.Sprint("Unknown zpool property: ",
				PoolPropertyToName(p)))
			return
		}
		var list C.property_list_t
		r := C.read_zpool_property(pool.list.zph, &list, C.int(p))
		if r != 0 {
			err = LastError()
		}
		prop.Value = C.GoString(&(list.value[0]))
		prop.Source = C.GoString(&(list.source[0]))
		pool.Properties[p] = prop
		return
	}
	return prop, errors.New(msgPoolIsNil)
}

// GetFeature reload and return single specified feature. This also reloads requested
// feature in Features map.
func (pool *Pool) GetFeature(name string) (value string, err error) {
	var fvalue [512]C.char
	csName := C.CString(fmt.Sprint("feature@", name))
	r := C.zpool_prop_get_feature(pool.list.zph, csName, &(fvalue[0]), 512)
	C.free(unsafe.Pointer(csName))
	if r != 0 {
		err = errors.New(fmt.Sprint("Unknown zpool feature: ", name))
		return
	}
	value = C.GoString(&(fvalue[0]))
	pool.Features[name] = value
	return
}

// SetProperty set ZFS pool property to value. Not all properties can be set,
// some can be set only at creation time and some are read only.
// Always check if returned error and its description.
func (pool *Pool) SetProperty(p Prop, value string) (err error) {
	if pool.list != nil {
		// First check if property exist at all
		if p < PoolPropName || p > PoolNumProps {
			err = errors.New(fmt.Sprint("Unknown zpool property: ",
				PoolPropertyToName(p)))
			return
		}
		csPropName := C.CString(PoolPropertyToName(p))
		csPropValue := C.CString(value)
		r := C.zpool_set_prop(pool.list.zph, csPropName, csPropValue)
		C.free(unsafe.Pointer(csPropName))
		C.free(unsafe.Pointer(csPropValue))
		if r != 0 {
			err = LastError()
		} else {
			// Update Properties member with change made
			if _, err = pool.GetProperty(p); err != nil {
				return
			}
		}
		return
	}
	return errors.New(msgPoolIsNil)
}

// Close ZFS pool handler and release associated memory.
// Do not use Pool object after this.
func (pool *Pool) Close() {
	C.zpool_list_close(pool.list)
	pool.list = nil
}

// Name get (re-read) ZFS pool name property
func (pool *Pool) Name() (name string, err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
	} else {
		name = C.GoString(C.zpool_get_name(pool.list.zph))
		pool.Properties[PoolPropName] = Property{Value: name, Source: "none"}
	}
	return
}

// State get ZFS pool state
// Return the state of the pool (ACTIVE or UNAVAILABLE)
func (pool *Pool) State() (state PoolState, err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
	} else {
		state = PoolState(C.zpool_read_state(pool.list.zph))
	}
	return
}

func (vdev *VDevTree) isGrouping() (grouping bool, mindevs, maxdevs int) {
	maxdevs = int(^uint(0) >> 1)
	if vdev.Type == VDevTypeRaidz {
		grouping = true
		if vdev.Parity == 0 {
			vdev.Parity = 1
		}
		if vdev.Parity > 254 {
			vdev.Parity = 254
		}
		mindevs = int(vdev.Parity) + 1
		maxdevs = 255
	} else if vdev.Type == VDevTypeMirror {
		grouping = true
		mindevs = 2
	} else if vdev.Type == VDevTypeLog || vdev.Type == VDevTypeSpare || vdev.Type == VDevTypeL2cache {
		grouping = true
		mindevs = 1
	}
	return
}

func (vdev *VDevTree) isLog() (r C.uint64_t) {
	r = 0
	if vdev.Type == VDevTypeLog {
		r = 1
	}
	return
}

func toCPoolProperties(props PoolProperties) (cprops *C.nvlist_t) {
	cprops = nil
	for prop, value := range props {
		name := C.zpool_prop_to_name(C.zpool_prop_t(prop))
		csPropValue := C.CString(value)
		r := C.add_prop_list(name, csPropValue, &cprops, C.boolean_t(1))
		C.free(unsafe.Pointer(csPropValue))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}
	return
}

func toCDatasetProperties(props DatasetProperties) (cprops *C.nvlist_t) {
	cprops = nil
	for prop, value := range props {
		name := C.zfs_prop_to_name(C.zfs_prop_t(prop))
		csPropValue := C.CString(value)
		r := C.add_prop_list(name, csPropValue, &cprops, C.boolean_t(0))
		C.free(unsafe.Pointer(csPropValue))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}
	return
}

func buildVDevTree(root *C.nvlist_t, rtype VDevType, vdevs []VDevTree,
	props PoolProperties) (err error) {
	count := len(vdevs)
	if count == 0 {
		return
	}
	childrens := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	defer C.nvlist_free_array(childrens)
	spares := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	nspares := 0
	defer C.nvlist_free_array(spares)
	l2cache := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	nl2cache := 0
	defer C.nvlist_free_array(l2cache)
	for i, vdev := range vdevs {
		grouping, mindevs, maxdevs := vdev.isGrouping()
		var child *C.nvlist_t
		// fmt.Println(vdev.Type)
		if r := C.nvlist_alloc(&child, C.NV_UNIQUE_NAME, 0); r != 0 {
			err = errors.New("Failed to allocate vdev")
			return
		}
		vcount := len(vdev.Devices)
		if vcount < mindevs || vcount > maxdevs {
			err = fmt.Errorf(
				"Invalid vdev specification: %s supports no less than %d or more than %d devices",
				vdev.Type, mindevs, maxdevs)
			return
		}
		csType := C.CString(string(vdev.Type))
		r := C.nvlist_add_string(child, C.sZPOOL_CONFIG_TYPE,
			csType)
		C.free(unsafe.Pointer(csType))
		if r != 0 {
			err = errors.New("Failed to set vdev type")
			return
		}
		if r := C.nvlist_add_uint64(child, C.sZPOOL_CONFIG_IS_LOG,
			vdev.isLog()); r != 0 {
			err = errors.New("Failed to allocate vdev (is_log)")
			return
		}
		if grouping {
			if vdev.Type == VDevTypeRaidz {
				r := C.nvlist_add_uint64(child,
					C.sZPOOL_CONFIG_NPARITY,
					C.uint64_t(mindevs-1))
				if r != 0 {
					err = errors.New("Failed to allocate vdev (parity)")
					return
				}
			}
			if err = buildVDevTree(child, vdev.Type, vdev.Devices,
				props); err != nil {
				return
			}
		} else {
			// if vdev.Type == VDevTypeDisk {
			if r := C.nvlist_add_uint64(child,
				C.sZPOOL_CONFIG_WHOLE_DISK, 1); r != 0 {
				err = errors.New("Failed to allocate vdev child (whdisk)")
				return
			}
			// }
			if len(vdev.Path) > 0 {
				csPath := C.CString(vdev.Path)
				r := C.nvlist_add_string(
					child, C.sZPOOL_CONFIG_PATH,
					csPath)
				C.free(unsafe.Pointer(csPath))
				if r != 0 {
					err = errors.New("Failed to allocate vdev child (type)")
					return
				}
				ashift, _ := strconv.Atoi(props[PoolPropAshift])
				if ashift > 0 {
					if r := C.nvlist_add_uint64(child,
						C.sZPOOL_CONFIG_ASHIFT,
						C.uint64_t(ashift)); r != 0 {
						err = errors.New("Failed to allocate vdev child (ashift)")
						return
					}
				}
			}
			if vdev.Type == VDevTypeSpare {
				C.nvlist_array_set(spares, C.int(nspares), child)
				nspares++
				count--
				continue
			} else if vdev.Type == VDevTypeL2cache {
				C.nvlist_array_set(l2cache, C.int(nl2cache), child)
				nl2cache++
				count--
				continue
			}
		}
		C.nvlist_array_set(childrens, C.int(i), child)
	}
	if count > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.sZPOOL_CONFIG_CHILDREN, childrens,
			C.uint_t(count)); r != 0 {
			err = errors.New("Failed to allocate vdev children")
			return
		}
		// fmt.Println("childs", root, count, rtype)
		// debug.PrintStack()
	}
	if nl2cache > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.sZPOOL_CONFIG_L2CACHE, l2cache,
			C.uint_t(nl2cache)); r != 0 {
			err = errors.New("Failed to allocate vdev cache")
			return
		}
	}
	if nspares > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.sZPOOL_CONFIG_SPARES, spares,
			C.uint_t(nspares)); r != 0 {
			err = errors.New("Failed to allocate vdev spare")
			return
		}
		// fmt.Println("spares", root, count)
	}
	return
}

// PoolCreate create ZFS pool per specs, features and properties of pool and root dataset
func PoolCreate(name string, vdevs []VDevTree, features map[string]string,
	props PoolProperties, fsprops DatasetProperties) (pool Pool, err error) {
	// create root vdev nvroot
	var nvroot *C.nvlist_t
	if r := C.nvlist_alloc(&nvroot, C.NV_UNIQUE_NAME, 0); r != 0 {
		err = errors.New("Failed to allocate root vdev")
		return
	}
	csTypeRoot := C.CString(string(VDevTypeRoot))
	r := C.nvlist_add_string(nvroot, C.sZPOOL_CONFIG_TYPE,
		csTypeRoot)
	C.free(unsafe.Pointer(csTypeRoot))
	if r != 0 {
		err = errors.New("Failed to allocate root vdev")
		return
	}
	defer C.nvlist_free(nvroot)

	// Now we need to build specs (vdev hierarchy)
	if err = buildVDevTree(nvroot, VDevTypeRoot, vdevs, props); err != nil {
		return
	}

	// convert properties
	cprops := toCPoolProperties(props)
	if cprops != nil {
		defer C.nvlist_free(cprops)
	} else if len(props) > 0 {
		err = errors.New("Failed to allocate pool properties")
		return
	}
	cfsprops := toCDatasetProperties(fsprops)
	if cfsprops != nil {
		defer C.nvlist_free(cfsprops)
	} else if len(fsprops) > 0 {
		err = errors.New("Failed to allocate FS properties")
		return
	}
	for fname, fval := range features {
		csName := C.CString(fmt.Sprintf("feature@%s", fname))
		csVal := C.CString(fval)
		r := C.add_prop_list(csName, csVal, &cprops,
			C.boolean_t(1))
		C.free(unsafe.Pointer(csName))
		C.free(unsafe.Pointer(csVal))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}

	// Create actual pool then open
	csName := C.CString(name)
	defer C.free(unsafe.Pointer(csName))
	if r := C.zpool_create(libzfsHandle, csName, nvroot,
		cprops, cfsprops); r != 0 {
		err = LastError()
		err = errors.New(err.Error() + " (zpool_create)")
		return
	}

	// Open created pool and return handle
	pool, err = PoolOpen(name)
	return
}

// Status get pool status. Let you check if pool healthy.
func (pool *Pool) Status() (status PoolStatus, err error) {
	var msgid *C.char
	var reason C.zpool_status_t
	var errata C.zpool_errata_t
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
		return
	}
	reason = C.zpool_get_status(pool.list.zph, &msgid, &errata)
	status = PoolStatus(reason)
	return
}

// Destroy the pool.  It is up to the caller to ensure that there are no
// datasets left in the pool. logStr is optional if specified it is
// appended to ZFS history
func (pool *Pool) Destroy(logStr string) (err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
		return
	}
	csLog := C.CString(logStr)
	defer C.free(unsafe.Pointer(csLog))
	retcode := C.zpool_destroy(pool.list.zph, csLog)
	if retcode != 0 {
		err = LastError()
	}
	return
}

// Export exports the pool from the system.
// Before exporting the pool, all datasets within the pool are unmounted.
// A pool can not be exported if it has a shared spare that is currently
// being used.
func (pool *Pool) Export(force bool, log string) (err error) {
	var forcet C.boolean_t
	if force {
		forcet = 1
	}
	csLog := C.CString(log)
	defer C.free(unsafe.Pointer(csLog))
	if rc := C.zpool_export(pool.list.zph, forcet, csLog); rc != 0 {
		err = LastError()
	}
	return
}

// ExportForce hard force export of the pool from the system.
func (pool *Pool) ExportForce(log string) (err error) {
	csLog := C.CString(log)
	defer C.free(unsafe.Pointer(csLog))
	if rc := C.zpool_export_force(pool.list.zph, csLog); rc != 0 {
		err = LastError()
	}
	return
}

// VDevTree - Fetch pool's current vdev tree configuration, state and stats
func (pool *Pool) VDevTree() (vdevs VDevTree, err error) {
	var nvroot *C.nvlist_t
	var poolName string
	config := C.zpool_get_config(pool.list.zph, nil)
	if config == nil {
		err = fmt.Errorf("Failed zpool_get_config")
		return
	}
	if C.nvlist_lookup_nvlist(config, C.sZPOOL_CONFIG_VDEV_TREE,
		&nvroot) != 0 {
		err = fmt.Errorf("Failed to fetch %s", C.ZPOOL_CONFIG_VDEV_TREE)
		return
	}
	if poolName, err = pool.Name(); err != nil {
		return
	}
	return poolGetConfig(poolName, nvroot)
}
