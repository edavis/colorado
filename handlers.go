package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
)

func (r *River) serveLog(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	var buffer bytes.Buffer
	for _, message := range r.Messages {
		fmt.Fprintf(&buffer, message)
	}
	buffer.WriteTo(w)
}

func (r *River) serveRiver(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	js := RiverJS{
		Metadata: map[string]string{
			"name":             r.Name,
			"title":            r.Title,
			"description":      r.Description,
			"aggregator":       "colorado v0.1",
			"aggregatorDocs":   "https://github.com/edavis/colorado",
			"docs":             "http://riverjs.org/",
			"ctBuilds":         fmt.Sprintf("%d", r.builds),
			"whenGMT":          nowGMT(),
			"whenLocal":        nowLocal(),
			"whenStartedGMT":   r.whenStartedGMT,
			"whenStartedLocal": r.whenStartedLocal,
		},
		UpdatedFeeds: map[string][]*UpdatedFeed{
			"updatedFeed": r.Updates,
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("  ", "  ")
	enc.SetEscapeHTML(false)

	fmt.Fprintf(w, "%s(", callbackName)
	enc.Encode(js)
	fmt.Fprint(w, ")")
}

func (r *River) serveIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fname := path.Join("templates", "river_index.html")
	tmpl, err := template.ParseFiles(fname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
