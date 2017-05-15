package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"path"
)

// RiverJS is the root JSON returned by /river.
type RiverJS struct {
	Metadata     map[string]string         `json:"metadata"`
	UpdatedFeeds map[string][]*UpdatedFeed `json:"updatedFeeds"`
}

// UpdatedFeed contains the feed that had updates.
type UpdatedFeed struct {
	URL         string             `json:"feedUrl"`
	Website     string             `json:"websiteUrl"`
	Title       string             `json:"feedTitle"`
	Description string             `json:"feedDescription"`
	LastUpdate  string             `json:"whenLastUpdate"`
	Items       []*UpdatedFeedItem `json:"item"`
}

// UpdatedFeedItem contains the items of the updated feed.
type UpdatedFeedItem struct {
	Body      string `json:"body"`
	PermaLink string `json:"permaLink"`
	PubDate   string `json:"pubDate"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Id        string `json:"id"`
}

type OPML struct {
	XMLName  xml.Name  `xml:"opml"`
	Version  string    `xml:"version,attr"`
	Title    string    `xml:"head>title"`
	Docs     string    `xml:"head>docs"`
	Outlines []Outline `xml:"body>outline"`
}

type Outline struct {
	Text string `xml:"text,attr"`
	Type string `xml:"type,attr"`
	URL  string `xml:"xmlUrl,attr"`
}

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
			"aggregator":       userAgent,
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

func (r *River) serveFeedsOpml(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")

	opml := OPML{
		Version: "2.0",
		Title:   r.Name + " feeds",
		Docs:    opmlDocs,
	}
	for _, webFeed := range r.Streams {
		outline := Outline{
			Text: webFeed.URL,
			URL:  webFeed.URL,
			Type: "rss",
		}
		opml.Outlines = append(opml.Outlines, outline)
	}

	encoded, err := xml.MarshalIndent(opml, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xml.Header))
	w.Write(encoded)
}
