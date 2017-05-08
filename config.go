package main

import (
	"github.com/naoina/toml"
	"os"
)

const (
	utcTimestampFmt   = "Mon, 02 Jan 2006 15:04:05 GMT"
	localTimestampFmt = "Mon, 02 Jan 2006 15:04:05 MST"
	callbackName      = "onGetRiverStream"
	maxEventLog       = 250
	maxCharCount      = 280
)

type Config struct {
	Settings struct {
		Update string
	}
	River []struct {
		Name        string
		Title       string
		Description string
		Feeds       []string
		Update      string
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
