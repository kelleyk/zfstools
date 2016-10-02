package gokk

import "regexp"

const (
	// RFC3339Pattern is a regular expression pattern that matches partial strings in the format given by
	// `time.RFC3339`, which is that specified in RFC 3339.  It contains no capturing groups.
	RFC3339Pattern = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|(?:[+-]\d{2}:\d{2}))?`
)

var (
	// RFC3339Regexp is a regular expression that matches whole strings in the format given by `time.RFC3339`, which is
	// that specified in RFC 3339.
	RFC3339Regexp = regexp.MustCompile(`(?i)^(` + RFC3339Pattern + `)$`)
)
