package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseSnapName(t *testing.T) {
	const prefix = "zfs-auto-snap"

	for _, tt := range []struct {
		path string
		meta *snapMetadata
	}{
		{"ds@zfs-auto-snap_daily_2010-01-02T03:04:05Z", &snapMetadata{dataset: "ds", label: "daily", ts: time.Date(2010, 1, 2, 3, 4, 5, 0, time.UTC)}},
		{"ds@some-other-prefix_daily_2010-01-02T03:04:05Z", nil},
	} {
		meta, err := parseSnapName(prefix, tt.path)

		if assert.Nil(t, err) {
			if tt.meta == nil {
				assert.Nil(t, meta, "did not expect name to match, but result was returned")
			} else {
				if assert.NotNil(t, meta, "expected name to match, but no result was returned") {
					ok := true
					ok = ok || assert.Equal(t, prefix, meta.prefix)
					ok = ok || assert.Equal(t, tt.meta.label, meta.label)
					ok = ok || assert.Equal(t, tt.meta.ts, meta.ts)
					ok = ok || assert.Equal(t, tt.meta.dataset, meta.dataset)

					// Finally, check that we can get the original snapshot path back.
					if ok {
						assert.Equal(t, tt.path, meta.Path())
					}
				}
			}
		}
	}
}
