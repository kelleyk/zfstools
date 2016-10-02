package zfs

/*
#cgo CFLAGS: -I /usr/include/libzfs -I /usr/include/libspl -DHAVE_IOCTL_IN_SYS_IOCTL_H
#cgo LDFLAGS: -lnvpair

#include <libnvpair.h>
*/
import "C"
import (
	"fmt"
	"syscall"
	"unsafe"
)

/*
nvpair.h: https://github.com/zfsonlinux/zfs/blob/master/include/sys/nvpair.h
*/

// NVPair corresponds to nvpair_t; it's only the header.
//
// typedef struct nvpair {
// 	int32_t nvp_size;	/- size of this nvpair -/
// int16_t	nvp_name_sz;	/- length of name string -/
// int16_t	nvp_reserve;	/- not used -/
// int32_t	nvp_value_elem;	/- number of elements for array types -/
// data_type_t nvp_type;	/- type of value -/
// /- name string -/
// /- aligned ptr array for string arrays -/
// /- aligned array of data for value -/
// } nvpair_t;
type NVPair C.nvpair_t

func (p *NVPair) Type() DataType {
	return DataType(p.nvp_type)
}

func (p *NVPair) Size() int {
	return -1
}

// Length returns the number of elements (if the value is of an array type).
func (p *NVPair) Length() int {
	return -1
}

func (p *NVPair) Name() string {
	// See definition of NVP_NAME: https://github.com/zfsonlinux/zfs/blob/master/include/sys/nvpair.h#L116
	namePtr := uintptr(unsafe.Pointer(p)) + C.sizeof_nvpair_t
	return C.GoString((*C.char)(unsafe.Pointer(namePtr)))
}

func (p *NVPair) ValueString() string {
	v := p.Value()
	switch v := v.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		panic("unable to convert value to string")
	}
}

// XXX: This function borrowed from nathan7/go-nvpair.
func (p *NVPair) Value() interface{} {
	n := p

	nc := (*C.nvpair_t)(n)
	var value interface{}
	var ret C.int

	switch n.Type() {
	case TypeBooleanValue:
		var v C.boolean_t
		ret = C.nvpair_value_boolean_value(nc, &v)
		value = copyinBool(v)
	case TypeBoolean:
		return true
	case TypeByte:
		var v C.uchar_t
		ret = C.nvpair_value_byte(nc, &v)
		value = byte(v)
	case TypeUint8:
		var v C.uint8_t
		ret = C.nvpair_value_uint8(nc, &v)
		value = uint8(v)
	case TypeInt8:
		var v C.int8_t
		ret = C.nvpair_value_int8(nc, &v)
		value = int8(v)
	case TypeUint16:
		var v C.uint16_t
		ret = C.nvpair_value_uint16(nc, &v)
		value = uint16(v)
	case TypeInt16:
		var v C.int16_t
		ret = C.nvpair_value_int16(nc, &v)
		value = int16(v)
	case TypeUint32:
		var v C.uint32_t
		ret = C.nvpair_value_uint32(nc, &v)
		value = uint32(v)
	case TypeInt32:
		var v C.int32_t
		ret = C.nvpair_value_int32(nc, &v)
		value = int32(v)
	case TypeUint64:
		var v C.uint64_t
		ret = C.nvpair_value_uint64(nc, &v)
		value = uint64(v)
	case TypeInt64:
		var v C.int64_t
		ret = C.nvpair_value_int64(nc, &v)
		value = int64(v)
	case TypeDouble:
		var v C.double
		ret = C.nvpair_value_double(nc, &v)
		value = float64(v)
	case TypeString:
		var v *C.char
		ret = C.nvpair_value_string(nc, &v)
		value = C.GoString(v)
	case TypeTime:
		var v C.hrtime_t
		ret = C.nvpair_value_hrtime(nc, &v)
		value = copyinTime(v)
	case TypeNVList:
		var v *C.nvlist_t
		ret = C.nvpair_value_nvlist(nc, &v)
		value = (*NVList)(v)
	case TypeBooleanArray:
		var (
			p *C.boolean_t
			n C.uint_t
		)
		if ret = C.nvpair_value_boolean_array(nc, &p, &n); ret == 0 {
			value = copyinBools(p, n)
		}
	case TypeByteArray:
		var (
			p *C.uchar_t
			n C.uint_t
		)
		if ret = C.nvpair_value_byte_array(nc, &p, &n); ret == 0 {
			value = copyinBytes(p, n)
		}
	case TypeUint8Array:
		var (
			p *C.uint8_t
			n C.uint_t
		)
		if ret = C.nvpair_value_uint8_array(nc, &p, &n); ret == 0 {
			value = copyinUint8s(p, n)
		}
	case TypeInt8Array:
		var (
			p *C.int8_t
			n C.uint_t
		)
		if ret = C.nvpair_value_int8_array(nc, &p, &n); ret == 0 {
			value = copyinInt8s(p, n)
		}
	case TypeUint16Array:
		var (
			p *C.uint16_t
			n C.uint_t
		)
		if ret = C.nvpair_value_uint16_array(nc, &p, &n); ret == 0 {
			value = copyinUint16s(p, n)
		}
	case TypeInt16Array:
		var (
			p *C.int16_t
			n C.uint_t
		)
		if ret = C.nvpair_value_int16_array(nc, &p, &n); ret == 0 {
			value = copyinInt16s(p, n)
		}

	case TypeUint32Array:
		var (
			p *C.uint32_t
			n C.uint_t
		)
		if ret = C.nvpair_value_uint32_array(nc, &p, &n); ret == 0 {
			value = copyinUint32s(p, n)
		}
	case TypeInt32Array:
		var (
			p *C.int32_t
			n C.uint_t
		)
		if ret = C.nvpair_value_int32_array(nc, &p, &n); ret == 0 {
			value = copyinInt32s(p, n)
		}
	case TypeUint64Array:
		var (
			p *C.uint64_t
			n C.uint_t
		)
		if ret = C.nvpair_value_uint64_array(nc, &p, &n); ret == 0 {
			value = copyinUint64s(p, n)
		}
	case TypeInt64Array:
		var (
			p *C.int64_t
			n C.uint_t
		)
		if ret = C.nvpair_value_int64_array(nc, &p, &n); ret == 0 {
			value = copyinInt64s(p, n)
		}
	case TypeStringArray:
		var (
			p **C.char
			n C.uint_t
		)
		if ret = C.nvpair_value_string_array(nc, &p, &n); ret == 0 {
			value = copyinStrings(p, n)
		}
	case TypeNVListArray:
		var (
			p **C.nvlist_t
			n C.uint_t
		)
		if ret = C.nvpair_value_nvlist_array(nc, &p, &n); ret == 0 {
			value = copyinNVLists(p, n)
		}
	}

	if ret != 0 {
		panic(syscall.Errno(ret))
	}
	return value

}
