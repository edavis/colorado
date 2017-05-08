package main

import (
	"fmt"
	"log"
	"net/http"
)

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
		indexUrl := fmt.Sprintf("/%s/", name)
		logUrl := fmt.Sprintf("/%s/log", name)
		riverUrl := fmt.Sprintf("/%s/river", name)
		mux.HandleFunc(indexUrl, river.serveIndex)
		mux.HandleFunc(logUrl, river.serveLog)
		mux.HandleFunc(riverUrl, river.serveRiver)

		// start fetching feeds
		go river.Run()
	}

	log.Fatalln(http.ListenAndServe(":9000", mux))
}
