package main

import (
	"fmt"
	"regexp"
	"time"
)

const (
	// `Mon Jan 2 15:04:05 -0700 MST 2006`
	snapNameTimestampFormat = `2006.01.02_15.04.05`
)

var (
	// dataset@zfs-auto-snap_YYYY.MM.DD_HHmm.SS
	snapNameRegexp = regexp.MustCompile(`^(.*)@(.+)_([^_]+)_(\d{4}\.\d{2}\.\d{2}_\d{2}\.\d{2}\.\d{2})$`)
)

type snapMetadata struct {
	dataset string
	prefix  string
	label   string
	ts      time.Time
}

func (m *snapMetadata) Path() string {
	return fmt.Sprintf("%s@%s_%s_%s", m.dataset, m.prefix, m.label, m.ts.Format(snapNameTimestampFormat))
}

func parseSnapName(expectedPrefix, path string) (*snapMetadata, error) {

	m := snapNameRegexp.FindStringSubmatch(path)
	if len(m) == 0 {
		// No regexp match.
		fmt.Printf("no regexp match")
		return nil, nil
	}
	dataset, snapPrefix, label, tsStr := m[1], m[2], m[3], m[4]

	if snapPrefix != expectedPrefix {
		// Wrong prefix; no match.
		fmt.Printf("bad prefix")
		return nil, nil
	}

	ts, err := time.Parse(snapNameTimestampFormat, tsStr)
	if err != nil {
		return nil, err
	}

	return &snapMetadata{
		dataset: dataset,
		prefix:  snapPrefix,
		label:   label,
		ts:      ts,
	}, nil
}
