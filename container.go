package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
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

	mux.HandleFunc("/errors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		fp, err := os.Open("error.log")
		if err != nil {
			errorLog.Println(err)
		}
		defer fp.Close()

		reader := bufio.NewReader(fp)
		reader.WriteTo(w)
	})

	for name, river := range rc.Rivers {
		// register HTTP handlers
		mux.HandleFunc(fmt.Sprintf("/%s/", name), river.serveIndex)
		mux.HandleFunc(fmt.Sprintf("/%s/river", name), river.serveRiver)
		mux.HandleFunc(fmt.Sprintf("/%s/feeds.opml", name), river.serveFeedsOpml)

		// start fetching feeds
		go river.Run()
	}

	if err := http.ListenAndServe(":9000", mux); err != nil {
		logger.Fatalln(err)
	}
}
