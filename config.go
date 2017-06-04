package main

import (
	"github.com/naoina/toml"
	"os"
	"time"
)

const (
	userAgent         = "colorado/0.1 (https://github.com/edavis/colorado)"
	utcTimestampFmt   = "Mon, 02 Jan 2006 15:04:05 GMT"
	localTimestampFmt = "Mon, 02 Jan 2006 15:04:05 MST"
	callbackName      = "onGetRiverStream"
	opmlDocs          = "http://dev.opml.org/spec2.html"
	maxEventLog       = 250
	maxCharCount      = 280
	maxItems          = 5
	maxFeedUpdates    = 100
	pollChange        = 0.1 // scale poll interval by this percentage
	pollDefault       = time.Duration(1 * time.Hour)
	pollMin           = time.Duration(5 * time.Minute)
	pollMax           = time.Duration(1 * time.Hour)
)

type Config struct {
	River []struct {
		Name        string
		Title       string
		Description string
		Feeds       []string
		OPML        string
	}
}

func loadConfig(path string) (*Config, error) {
	var config Config

	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	if err := toml.NewDecoder(fp).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
