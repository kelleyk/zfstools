package main

import (
	"fmt"

	"github.com/kelleyk/go-libzfs"
)

func main() {
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

		fmt.Printf("%v\n  state: %v\n  status: %v\n\n", name, state, status)
	}
}
