package main

import "encoding/xml"

type OPML struct {
	XMLName  xml.Name  `xml:"opml"`
	Version  string    `xml:"version,attr"`
	Title    string    `xml:"head>title"`
	Docs     string    `xml:"head>docs"`
	Outlines []Outline `xml:"body>outline"`
}

type Outline struct {
	Text     string    `xml:"text,attr"`
	Type     string    `xml:"type,attr"`
	URL      string    `xml:"xmlUrl,attr"`
	Interval string    `xml:"pollInterval,attr"`
	Outlines []Outline `xml:"outline"`
}

func _extract(outlines []Outline, feeds *[]string) []string {
	for _, outline := range outlines {
		switch {
		case outline.URL != "":
			*feeds = append(*feeds, outline.URL)
		case len(outline.Outlines) > 0:
			_extract(outline.Outlines, feeds)
		}
	}
	return *feeds
}

func (root OPML) urls() []string {
	var feeds []string
	return _extract(root.Outlines, &feeds)
}
