package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func (r *River) serveLog(w http.ResponseWriter, req *http.Request) {
	b := r.buffer.Bytes()
	w.Write(b)
	r.buffer = bytes.NewBuffer(b)
	r.Logger.SetOutput(r.buffer)
}

func (r *River) serveRiver(w http.ResponseWriter, req *http.Request) {
	js := RiverJS{
		Metadata: map[string]string{
			"name":             r.Name,
			"aggregator":       "colorado v0.1",
			"aggregatorDocs":   "https://github.com/edavis/colorado",
			"docs":             "http://riverjs.org/",
			"ctBuilds":         fmt.Sprintf("%d", r.builds),
			"whenGMT":          nowGMT(),
			"whenLocal":        nowLocal(),
			"whenStartedGMT":   r.whenStartedGMT,
			"whenStartedLocal": r.whenStartedLocal,
		},
	}
	js.UpdatedFeeds.UpdatedFeed = r.Updates

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	fmt.Fprintf(w, "%s(", callbackName)
	enc.Encode(js)
	fmt.Fprint(w, ")")
}
