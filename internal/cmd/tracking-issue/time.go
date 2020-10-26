package main

import (
	"fmt"
	"time"
)

// now returns the current time for relative formatting. This is overwritten
// during tests to ensure that our output can be byte-for-byte compared against
// the golden output file.
var now = time.Now

// formatTimeSince will return a string containing the number of days since the
// given time.
func formatTimeSince(t time.Time) string {
	days := now().UTC().Sub(t.UTC()) / time.Hour / 24

	switch days {
	case 0:
		return "today"
	case 1:
		return "1 day ago"
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}
