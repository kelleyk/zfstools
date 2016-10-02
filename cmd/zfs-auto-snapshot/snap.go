package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/kelleyk/gokk"
)

const (
	// `Mon Jan 2 15:04:05 -0700 MST 2006`
	snapNameTimestampFormat = time.RFC3339
)

var (
	// dataset@zfs-auto-snap_label_ts
	//   where ts format = e.g. `2006-01-02T15:04:05Z07:00`
	snapNameRegexp = regexp.MustCompile(`(?i)^(.*)@(.+)_([^_]+)_(` + gokk.RFC3339Pattern + `)$`)
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
		return nil, nil
	}
	dataset, snapPrefix, label, tsStr := m[1], m[2], m[3], m[4]

	if snapPrefix != expectedPrefix {
		// Wrong prefix; no match.
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

type byTS []*snapMetadata

func (a byTS) Len() int           { return len(a) }
func (a byTS) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTS) Less(i, j int) bool { return a[i].ts.After(a[j].ts) }
