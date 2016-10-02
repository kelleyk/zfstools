package main

import zfs "github.com/kelleyk/go-libzfs"

// walkDataset visits the dataset d and its children, including snapshots.
//
// TODO: move to go-libzfs
//
func walkDataset(f func(zfs.Dataset) error, d zfs.Dataset) error {
	if err := f(d); err != nil {
		return err
	}

	for _, child := range d.Children {
		if err := walkDataset(f, child); err != nil {
			return err
		}
	}
	return nil
}

// poolScanning returns true iff the given dataset is on a pool that has a scan (i.e. a scrub or resilver) in progress
func poolScanning(d zfs.Dataset) (bool, error) {
	p, err := d.Pool()
	if err != nil {
		return false, err
	}
	rootVDev, err := p.VDevTree()
	if err != nil {
		return false, err
	}
	return (rootVDev.ScanStat.State == zfs.DSLScanStateScanning), nil
}
