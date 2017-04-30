package main

import "time"

func nowGMT() string {
	return time.Now().UTC().Format(utcTimestampFmt)
}

func nowLocal() string {
	return time.Now().Format(localTimestampFmt)
}
