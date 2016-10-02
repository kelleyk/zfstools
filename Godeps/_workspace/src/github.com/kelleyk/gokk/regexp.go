package gokk

import "regexp"

const (
	RFC3339Pattern = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|(?:[+-]\d{2}:\d{2}))?`
)

var (
	// dataset@zfs-auto-snap_label_ts
	//   where ts format = e.g. `2006-01-02T15:04:05Z07:00`
	RFC3339Regexp = regexp.MustCompile(`(?i)^(` + RFC3339Pattern + `)$`)
)
