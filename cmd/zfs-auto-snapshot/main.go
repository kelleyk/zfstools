package main

import (
	"errors"
	"flag"
	"fmt"

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
var (
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
	flag.Parse()

	l := logrus.New()
	_ = l
	// TODO: set up logger

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

/*
// Return value:
// - true iff the dataset has been deliberately selected
// - true iff we *would* snapshot this but are skipping it for some temporary reason (like e.g. scrubbing)
// - error
func (tool *Tool) datasetExcluded() (bool, bool, error) {
	if *skipScrub {
		scanning, err := poolScanning(d)
		if err != nil {
			return false, false, err
		}
		if scanning {
			// xxx...
			continue
		}
	}
}

func (tool *Tool) targetDatasets() ([]*zfs.Dataset, error) {

	ds, err := getDatasetsByName()
	if err != nil {
		return err
	}
	for name, d := range ds {
		if !tool.datasetExcluded(d) {
			targetDatasets = append(targetDatasets, d)
		}
	}

}

func (tool *Tool) allDatasets() ([]*zfs.Dataset, error) {

}
*/

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
		walkDataset(func(dd zfs.Dataset) error {
			if dd.Properties[zfs.DatasetPropType].Value == "snapshot" {
				return nil
			}
			path, err := dd.Path()
			if err != nil {
				return err
			}
			tool.datasetsByName[path] = dd
			return nil
		}, d)
	}

	return nil

}

func (tool *Tool) Main() error {
	defer tool.cleanup()
	if err := tool.preinit(); err != nil {
		return err
	}

	l := tool.l

	listPools()

	targetDatasets := make(map[string]zfs.Dataset)
	if len(flag.Args()) == 0 {
		return errors.New("filesystem argument list is empty")
	}
	if len(flag.Args()) == 1 && flag.Arg(0) == "//" {
		// TODO: If -recursive given, show warning that it is not necessary?
		// apply -default-exclude
		for path, d := range tool.datasetsByName {
			targetDatasets[path] = d
		}
		// "//" is a special value that means "all datasets" (subject to other constraints).
	} else {
		// show warning/error on -default-exclude here

		for _, dArg := range flag.Args() {
			if dArg == "//" {
				return errors.New("the // must be the only argument if it is given")
			}
			d, ok := tool.datasetsByName[dArg]
			if !ok {
				return fmt.Errorf("no such dataset: %v", dArg)
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
					return err
				}
			} else {
				path, err := d.Path()
				if err != nil {
					return err
				}
				targetDatasets[path] = d
			}
		}
	}

	for path, d := range targetDatasets {
		// // apply default-exclude policy
		// if *defaultExclude {
		// 	// exclude any datasets that do not have "com.sun:auto-snapshot" set.
		// 	for propID, v := range d.Properties {
		// 		propName := zfs.DatasetPropertyToName(propID)
		// 		log.Printf("  (%v) %v = %v", propID, propName, v.Value)
		// 	}
		// 	log.Printf("...")
		// }

		// apply skip-scrub to everything
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

	for dsName, d := range targetDatasets {
		fmt.Printf(" - %s\n", dsName)

		if err := d.VisitProperties(func(propID zfs.Prop, propName string, prop zfs.Property) error {
			fmt.Printf("   - %v = %v [%v]\n", propName, prop.Value, prop.Source)
			return nil
		}); err != nil {
			return err
		}
	}

	/*
		targetDatasets, err = tool.targetDatasets()

		for _, d := range targetDatasets {
			scan, err := poolScanning(d)
			if err != nil {
				return err
			}

			if scan {
				panic("scan")
			}
		}
	*/

	// the script then:
	// - ZPOOL_STATUS=$(zpool status)
	// - ZFS_LIST becomes a table of datasets and  the "com.sun:auto-snapshot" and  "com.sun:auto-snapshot:daily" (where daily is the -label) properties

	/*
		if [ -n "$opt_fast_zfs_list" ]
		then
		SNAPSHOTS_OLD=$(env LC_ALL=C zfs list -H -t snapshot -o name -s name|grep $opt_prefix |awk '{ print substr( $0, length($0) - 14, length($0) ) " " $0}' |sort -r -k1,1 -k2,2|awk '{ print substr( $0, 17, length($0) )}') \
		|| { print_log error "zfs list $?: $SNAPSHOTS_OLD"; exit 137; }
		else
		SNAPSHOTS_OLD=$(env LC_ALL=C zfs list -H -t snapshot -S creation -o name) \
		|| { print_log error "zfs list $?: $SNAPSHOTS_OLD"; exit 137; }
		fi

	*/

	// listDatasets()

	// walkVdevs()

	return nil
}
