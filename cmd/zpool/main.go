package main

import (
	"fmt"

	zfs "github.com/kelleyk/go-libzfs"
)

func listPools() {
	pools, err := zfs.PoolOpenAll()
	if err != nil {
		panic(err)
	}

	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()

	for _, p := range pools {
		name, err := p.Name()
		if err != nil {
			panic(err)
		}

		state, err := p.State()
		if err != nil {
			panic(err)
		}

		status, err := p.Status()
		if err != nil {
			panic(err)
		}

		vdevTree, err := p.VDevTree()
		if err != nil {
			panic(err)
		}

		fmt.Printf("%v\n  state: %v\n  status: %v\n", name, state, status)
		fmt.Printf("  root-vdev stat-state: %s\n", vdevTree.Stat.State)
		fmt.Printf("  root-vdev scanstat-state: %s\n", vdevTree.ScanStat.State)
		fmt.Printf("\n")
	}
}

func main() {

}
