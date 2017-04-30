package main

import (
	"github.com/microcosm-cc/bluemonday"
	"time"
)

func nowGMT() string {
	return time.Now().UTC().Format(utcTimestampFmt)
}

func nowLocal() string {
	return time.Now().Format(localTimestampFmt)
}

func sanitizeDate(date string) string {
	formats := []string{
		"Mon, 02 Jan 2006 15:04:05 UTC",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -07:00",
		"Mon, 2 Jan 2006 15:04:05 UTC",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -07:00",
		"02 Jan 2006 15:04:05 UTC",
		"02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 -07:00",
		"2 Jan 2006 15:04:05 UTC",
		"2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05-0700",
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, date); err == nil {
			return parsed.Format(utcTimestampFmt)
		}
	}

	return time.Now().UTC().Format(utcTimestampFmt)
}

func makePlainText(s string) string {
	p := bluemonday.StrictPolicy()
	return p.Sanitize(s)
}
