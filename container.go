package main

import (
	"fmt"
	"net/http"
)

type RiverContainer struct {
	Rivers map[string]*River
}

func NewRiverContainer(config *Config) *RiverContainer {
	rc := RiverContainer{
		Rivers: make(map[string]*River),
	}

	for _, obj := range config.River {
		var interval string

		switch {
		case obj.Update != "":
			interval = obj.Update
		case config.Settings.Update != "":
			interval = config.Settings.Update
		default:
			interval = "15m"
		}

		rc.Rivers[obj.Name] = NewRiver(obj.Name, obj.Feeds, interval, obj.Title, obj.Description)
	}

	return &rc
}

func (rc *RiverContainer) Run() {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	for name, river := range rc.Rivers {
		// register HTTP handlers
		mux.HandleFunc(fmt.Sprintf("/%s/", name), river.serveIndex)
		mux.HandleFunc(fmt.Sprintf("/%s/log", name), river.serveLog)
		mux.HandleFunc(fmt.Sprintf("/%s/river", name), river.serveRiver)
		mux.HandleFunc(fmt.Sprintf("/%s/feeds.opml", name), river.serveFeedsOpml)

		// start fetching feeds
		go river.Run()
	}

	if err := http.ListenAndServe(":9000", mux); err != nil {
		logger.Fatalln(err)
	}
}
