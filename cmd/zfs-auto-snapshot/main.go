package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kelleyk/go-libzfs"
)

/*

echo "Usage: $0 [options] [-l label] <'//' | name [name...]>
  --default-exclude  Exclude datasets if com.sun:auto-snapshot is unset.
  -d, --debug        Print debugging messages.
  -e, --event=EVENT  Set the com.sun:auto-snapshot-desc property to EVENT.
      --fast         Use a faster zfs list invocation.
  -n, --dry-run      Print actions without actually doing anything.
  -s, --skip-scrub   Do not snapshot filesystems in scrubbing pools.
  -h, --help         Print this usage message.
  -k, --keep=NUM     Keep NUM recent snapshots and destroy older snapshots.
  -l, --label=LAB    LAB is usually 'hourly', 'daily', or 'monthly'.
  -p, --prefix=PRE   PRE is 'zfs-auto-snap' by default.
  -q, --quiet        Suppress warnings and notices at the console.
      --send-full=F  Send zfs full backup. Unimplemented.
      --send-incr=F  Send zfs incremental backup. Unimplemented.
      --sep=CHAR     Use CHAR to separate date stamps in snapshot names.
  -g, --syslog       Write messages into the system log.
  -r, --recursive    Snapshot named filesystem and all descendants.
  -v, --verbose      Print info messages.
      --destroy-only Only destroy older snapshots, do not create new ones.
      name           Filesystem and volume names, or '//' for all ZFS datasets.
"
*/

const (
	// AutoSnapshotProperty is the name of a property that can be attached to datasets in order to indicate whether they
	// should be explicitly included or excluded from automatic snapshots.  When a value is not present, the dataset
	// will be included if -default-exclude is not given, and excluded if it is.
	//
	// N.B.: user properties are *always* strings; they can be up to 1024 characters.
	//
	AutoSnapshotProperty = "com.sun:auto-snapshot"
)

var (
	logLevel = flag.String("log-level", "WARN", "XXX: write usage string")

	help        = flag.Bool("help", false, "Print this usage message.")
	dryRun      = flag.Bool("dry-run", false, "Print actions without actually doing anything.")
	destroyOnly = flag.Bool("destroy-only", false, "Only destroy older snapshots, do not create new ones.")

	keep = flag.Int64("keep", 0, "Keep NUM recent snapshots and destroy older snapshots.")

	label = flag.String("label", "", "LAB is usually 'hourly', 'daily', or 'monthly'.")
	event = flag.String("event", "", "Set the com.sun:auto-snapshot-desc property to EVENT.")

	recursive      = flag.Bool("recursive", false, "Snapshot named filesystem and all descendants.")
	defaultExclude = flag.Bool("default-exclude", false, "Exclude datasets if com.sun:auto-snapshot is unset.")
	skipScrub      = flag.Bool("skip-scrub", true, "Do not snapshot filesystems in scrubbing pools.") // XXX: skip-scan instead?

	debug   = flag.Bool("default", false, "Print debugging messages.")
	quiet   = flag.Bool("quiet", false, "Suppress warnings and notices at the console.")
	syslog  = flag.Bool("syslog", false, "Write messages into the system log.")
	verbose = flag.Bool("verbose", false, "Print info messages.")
	prefix  = flag.String("prefix", "zfs-auto-snap", "")

	// send-full, send-incr, sep
)

func main() {
	var err error

	flag.Parse()

	l := logrus.New()
	l.Level, err = logrus.ParseLevel(*logLevel)
	if err != nil {
		l.Fatal("failed to parse -log-level")
	}

	if *help {
		// TODO: add to usage:
		//    Filesystem and volume names, or '//' for all ZFS datasets.
		flag.Usage()
		return
	}

	tool := &Tool{l: l}
	if err := tool.Main(); err != nil {
		l.WithError(err).Fatal()
	}
}

type Tool struct {
	l *logrus.Logger

	rootDatasets   []zfs.Dataset
	datasetsByName map[string]zfs.Dataset
}

func (tool *Tool) cleanup() {
	defer func() {
		for _, d := range tool.rootDatasets {
			d.Close()
		}
	}()

}

func (tool *Tool) preinit() error {
	var err error

	tool.datasetsByName = make(map[string]zfs.Dataset)
	tool.rootDatasets, err = zfs.DatasetOpenAll()
	if err != nil {
		panic(err)
	}

	for _, d := range tool.rootDatasets {
		if err := walkDataset(func(dd zfs.Dataset) error {
			if dd.Properties[zfs.DatasetPropType].Value == "snapshot" {
				return nil
			}
			path, err := dd.Path()
			if err != nil {
				return err
			}
			tool.datasetsByName[path] = dd
			return nil
		}, d); err != nil {
			return nil
		}
	}

	return nil

}

func (tool *Tool) selectDatasets(names []string) (map[string]zfs.Dataset, error) {

	targetDatasets := make(map[string]zfs.Dataset)

	if len(names) == 0 {
		return nil, errors.New("filesystem argument list is empty")
	}
	if len(names) == 1 && names[0] == "//" {
		// TODO: If -recursive given, show warning that it is not necessary?
		// apply -default-exclude
		for path, d := range tool.datasetsByName {
			targetDatasets[path] = d
		}
		// "//" is a special value that means "all datasets" (subject to other constraints).
	} else {
		// show warning/error on -default-exclude here

		for _, dArg := range names {
			if dArg == "//" {
				return nil, errors.New("the // must be the only argument if it is given")
			}
			d, ok := tool.datasetsByName[dArg]
			if !ok {
				return nil, fmt.Errorf("no such dataset: %v", dArg)
			}
			if *recursive {
				if err := walkDataset(func(dd zfs.Dataset) error {
					if dd.Properties[zfs.DatasetPropType].Value == "snapshot" {
						return nil
					}
					path, err := dd.Path()
					if err != nil {
						return err
					}
					targetDatasets[path] = dd
					return nil
				}, d); err != nil {
					return nil, err
				}
			} else {
				path, err := d.Path()
				if err != nil {
					return nil, err
				}
				targetDatasets[path] = d
			}
		}
	}

	return targetDatasets, nil
}

func (tool *Tool) getDatasetExcluded(d zfs.Dataset, defaultExclude bool) (bool, error) {
	l := tool.l

	dPath, err := d.Path()
	if err != nil {
		return false, err
	}

	prop, ok := d.UserProperties[AutoSnapshotProperty]
	if !ok {
		return defaultExclude, nil
	}

	switch strings.ToLower(prop.Value) {
	case "true":
		return false, nil
	case "false":
		return true, nil
	default:
		l.WithFields(logrus.Fields{"dataset": dPath}).Warnf("unexpected value for property: %s", AutoSnapshotProperty)
		return defaultExclude, nil
	}
}

func (tool *Tool) performSnapshots(d zfs.Dataset) error {
	// Get existing snapshots.
	if err := visitSnapshots(func(dd zfs.Dataset) error {

		path, err := dd.Path()
		if err != nil {
			return err
		}

		meta, err := parseSnapName(*prefix, path)
		if err != nil {
			return err
		}

		if meta != nil {
			// It's one of our auto snapshots!
			log.Printf("  label=%q  ts=%q", meta.label, meta.ts)
		}

		return nil
	}, d); err != nil {
		return err
	}

	// ... for each configured interval, see if the interval has been exceeded ...

	return nil
}

func (tool *Tool) Main() error {

	defer tool.cleanup()
	if err := tool.preinit(); err != nil {
		return err
	}

	l := tool.l

	targetDatasets, err := tool.selectDatasets(flag.Args())
	if err != nil {
		return err
	}

	for path, d := range targetDatasets {
		// Exclude datasets based on configuration properties and flags.
		exclude, err := tool.getDatasetExcluded(d, *defaultExclude)
		if err != nil {
			return err
		}
		if exclude {
			l.WithFields(logrus.Fields{"dataset": path}).Debug("excluded")
			delete(targetDatasets, path)
			continue
		} else {
			l.WithFields(logrus.Fields{"dataset": path}).Debug("not excluded")
		}

		// Exclude datasets that are on pools that are being scanned (e.g. scrubbed or resilvered).
		if *skipScrub {
			scanning, err := poolScanning(d)
			if err != nil {
				return err
			}
			if scanning {
				l.WithFields(logrus.Fields{"dataset": path}).Info("dataset skipped due to scan in progress")
				delete(targetDatasets, path)
			}
		}
	}

	for _, d := range targetDatasets {
		if err := tool.performSnapshots(d); err != nil {
			return err
		}
	}

	return nil
}
