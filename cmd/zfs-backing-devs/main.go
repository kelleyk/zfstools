package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	zfs "github.com/kelleyk/go-libzfs"
)

var (
	help = flag.Bool("help", false, "Print this usage message.")
)

func main() {
	var err error

	flag.Parse()

	if *help || len(flag.Args()) != 1 {
		flag.Usage()
		return
	}

	devs, err := getBackingDevices(flag.Arg(0))
	if err == nil && len(devs) == 0 {
		err = errors.New("failed to find any backing devices for dataset")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	for _, dev := range devs {
		fmt.Printf("%s\n", dev)
	}
}

func getBackingDevices(datasetPath string) ([]string, error) {
	ds, err := zfs.DatasetOpen(datasetPath)
	if err != nil {
		return []string{}, err
	}

	pool, err := ds.Pool()
	if err != nil {
		return []string{}, err
	}

	vdevTree, err := pool.VDevTree()
	if err != nil {
		return []string{}, err
	}

	var backingDevices []string
	if err := visitVDevTreeNodes(func(vdev *zfs.VDevTree) (bool, error) {
		switch vdev.Type {
		case zfs.VDevTypeRoot, zfs.VDevTypeMirror, zfs.VDevTypeRaidz:
			if len(vdev.Devices) == 0 {
				panic("expected device to have children")
			}
			return true, nil // recurse
		case zfs.VDevTypeHole:
			if len(vdev.Devices) > 0 {
				panic("did not expect device to have children")
			}
			return false, nil // ignore
		case zfs.VDevTypeSpare:
			if len(vdev.Devices) > 0 {
				panic("did not expect device to have children")
			}
			return true, nil // ignore, or include? (add flag?)
		case zfs.VDevTypeLog, zfs.VDevTypeL2cache:
			if len(vdev.Devices) > 0 {
				panic("did not expect device to have children")
			}
			return true, nil // ignore, or include? (add flag?)
		case zfs.VDevTypeDisk:
			// vdev.Path is the empty string; the name here is `/dev/mapper/d0-main_crypt`, which I bet is just the
			// naame that ZFS has for the device.
			backingDevices = append(backingDevices, vdev.Name)
			if len(vdev.Devices) > 0 {
				panic("did not expect device to have children")
			}
			return true, nil // backing things; but what to do with files?
		case zfs.VDevTypeFile:
			// XXX: Ideally, we'd probably figure out what device the file is on.
			panic("pool contains backing file; unsure what to do")

			// backingDevices = append(backingDevices, vdev.Path) // XXX: I'm just guessing on this one.
			// if len(vdev.Devices) > 0 {
			// 	panic("did not expect device to have children")
			// }
			// return true, nil // backing things; but what to do with files?
		case zfs.VDevTypeReplacing:
			if len(vdev.Devices) == 0 {
				panic("expected device to have children")
			}
			return true, nil // not sure what to do with this
		case zfs.VDevTypeMissing:
			if len(vdev.Devices) > 0 {
				panic("did not expect device to have children")
			}
			return true, nil // not sure what to do with this
		default:
			panic("unexpected vdev type")
		}
	}, &vdevTree); err != nil {
		return []string{}, err
	}

	return backingDevices, nil
}

func visitVDevTreeNodes(f func(*zfs.VDevTree) (bool, error), n *zfs.VDevTree) error {
	recurse, err := f(n)
	if err != nil {
		return err
	}

	if recurse {
		for _, child := range n.Devices {
			if err := visitVDevTreeNodes(f, &child); err != nil {
				return err
			}
		}
	}
	return nil
}
