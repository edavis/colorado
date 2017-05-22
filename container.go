package main

import (
	"bufio"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"html/template"
	"net/http"
	"os"
	"path"
)

type RiverContainer struct {
	Rivers map[string]*River
}

func NewRiverContainer(config *Config) *RiverContainer {
	rc := RiverContainer{
		Rivers: make(map[string]*River),
	}

	for _, obj := range config.River {
		var (
			feeds []string
			err   error
		)

		switch {
		case len(obj.Feeds) > 0:
			feeds = obj.Feeds
		case obj.OPML != "":
			feeds, err = extractFeedsFromOPML(obj.OPML)
			if err != nil {
				logger.Printf("couldn't get feeds from %s (%v)", obj.OPML, err)
			}
		}

		rc.Rivers[obj.Name] = NewRiver(obj.Name, obj.Title, obj.Description, feeds)
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
			errorLog.Printf("couldn't open error.log (%v)", err)
		}
		defer fp.Close()

		reader := bufio.NewReader(fp)
		reader.WriteTo(w)
	})

	// Set up the index handler
	mux.Handle("/", rc)

	if quickStart {
		logger.Println("quick start requested, skipping initial feed check")
	}

	for name, river := range rc.Rivers {
		// register HTTP handlers
		mux.HandleFunc(fmt.Sprintf("/%s/", name), river.serveIndex)
		mux.HandleFunc(fmt.Sprintf("/%s/river", name), river.serveRiver)
		mux.HandleFunc(fmt.Sprintf("/%s/feeds.opml", name), river.serveFeedsOpml)

		// start fetching feeds
		go river.Run()
	}

	go rc.Monitor()

	if err := http.ListenAndServe(":9000", mux); err != nil {
		logger.Fatalln(err)
	}
}

func (rc *RiverContainer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	t, err := template.ParseFiles(path.Join("templates", "index.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := t.Execute(w, rc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Monitor responds to watcher events and errors.
func (rc *RiverContainer) Monitor() {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op == fsnotify.Write {
				logger.Println("config file updated, adding/removing feeds as needed")
				if err := rc.UpdateRivers(); err != nil {
					logger.Printf("error updating rivers (%v)", err)
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				errorLog.Printf("received watcher error (%v)", err)
			}
		}
	}
}

// UpdateRivers is called when the config file is updated.
func (rc *RiverContainer) UpdateRivers() error {
	config, err := loadConfig(configPath)
	if err != nil {
		logger.Fatalln(err)
		return err
	}

	for _, obj := range config.River {
		if len(obj.Feeds) == 0 {
			continue
		}

		newFeeds := make(map[string]bool)
		river := rc.Rivers[obj.Name]

		// Add any new feeds to river.Streams
		for _, feed := range obj.Feeds {
			newFeeds[feed] = true

			if _, ok := river.Streams[feed]; !ok {
				logger.Printf("adding %q to %s river", feed, river.Name)
				river.Streams[feed] = true
				river.Updater <- feed
			}
		}

		// Remove any feeds in river.Streams that are no longer in the config file
		for feed, _ := range river.Streams {
			if _, ok := newFeeds[feed]; !ok {
				logger.Printf("removing %q from %s river", feed, river.Name)
				if stopped := river.Timers[feed].Stop(); !stopped {
					logger.Printf("problem stopping timer for %q", feed)
				}
				delete(river.Timers, feed)
				delete(river.Streams, feed)
			}
		}

	}

	return nil
}
