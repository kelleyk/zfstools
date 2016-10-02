# zfstools

This repository contains command-line utilities for working with ZFS.  They are written in Go.

They were developed against `zfsonlinux`, but should work (possibly with small adjustments) with other ZFS
implementations.

In order to build them, you will need `libzfs` and its dependencies (`libnvpair`, etc.).

## `zfs-auto-snapshot`

`zfs-auto-snapshot` is a Go analog of the `zfs-auto-snapshot` shell script that is distributed by the `zfsonlinux` project
(and others).

It is *not* a direct port.  The principal difference is that the tool reads a configuration file specifying snapshot
series, whereas that script has to be invoked once per snapshot series.

Configuration examples are available in the `cmd/zfs-auto-snapshot/_examples` subdirectory; once you have a
configuration file that you like, invoke the tool as follows.

    $ zfs-auto-snapshot -config=/path/to/config.yaml //

This will snapshot all datasets in all active pools.  You can specify individual dataset names in place of `//` if you
prefer; `-recursive` will also take snapshots of the children of named datasets.

You can mark specific datasets by setting a property on them.

    $ zfs set com.sun:auto-snapshot=false poolname/foo/bar

By default, a snapshot is taken of any selected dataset that does not have this property explicitly set to `false`.  If
`-default-exclude` is given, snapshots are only taken of those selected datasets that have it explicitly set to `true`.

If you'd like verbose output, try adding the `-log-level=INFO` option; for maximum verbosity, use `-log-level=DEBUG`.
