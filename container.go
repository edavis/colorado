package main

import (
	"fmt"
	"log"
	"net/http"
)

func NewRiverContainer() *RiverContainer {
	return &RiverContainer{
		Rivers: make(map[string]*River),
	}
}

func (rc *RiverContainer) Add(name string, feeds []string) {
	rc.Rivers[name] = NewRiver(name, feeds)
}

func (rc *RiverContainer) Run() {
	mux := http.NewServeMux()

	for name, river := range rc.Rivers {
		// register HTTP handlers
		logUrl := fmt.Sprintf("/%s/log", name)
		riverUrl := fmt.Sprintf("/%s/river", name)
		mux.HandleFunc(logUrl, river.serveLog)
		mux.HandleFunc(riverUrl, river.serveRiver)

		// start fetching feeds
		go river.Run()
	}

	log.Fatalln(http.ListenAndServe(":9000", mux))
}
