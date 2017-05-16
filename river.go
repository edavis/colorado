package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"net/http"
	"sync/atomic"
	"time"
)

type WebFeed struct {
	URL          string
	LastModified string
	ETag         string
}

// River holds the main app logic.
type River struct {
	Name             string
	Title            string
	Description      string
	FetchResults     chan FetchResult
	Streams          []*WebFeed
	Updates          []*UpdatedFeed
	Seen             map[string]bool
	UpdateInterval   time.Duration
	Messages         []string
	builds           uint64
	counter          uint64 // Item id counter
	httpClient       *http.Client
	whenStartedGMT   string // Track startup times
	whenStartedLocal string
}

// FetchResult holds the URL of the feed and its parsed representation.
type FetchResult struct {
	URL  string
	Feed *gofeed.Feed
}

func NewRiver(name string, feeds []string, updateInterval, title, description string) *River {
	r := River{
		Name:             name,
		Title:            title,
		Description:      description,
		FetchResults:     make(chan FetchResult),
		Seen:             make(map[string]bool),
		whenStartedGMT:   nowGMT(),
		whenStartedLocal: nowLocal(),
		httpClient:       http.DefaultClient,
	}

	duration, err := time.ParseDuration(updateInterval)
	if err != nil {
		errorLog.Printf("the duration %q is invalid, using default of 15 minutes", updateInterval)
		duration = 15 * time.Minute
	}
	r.UpdateInterval = duration

	for _, feed := range feeds {
		wf := WebFeed{URL: feed}
		r.Streams = append(r.Streams, &wf)
	}
	return &r
}

func (r *River) Run() {
	logger.Println("updating feeds (initial fetch)")
	r.UpdateFeeds()

	logger.Printf("fetching feeds every %s", r.UpdateInterval)
	ticker := time.NewTicker(r.UpdateInterval)

	for {
		select {
		case result := <-r.FetchResults:
			r.ProcessFeed(result)
		case <-ticker.C:
			logger.Println("updating feeds")
			r.UpdateFeeds()
		}
	}
}

func (r River) GetCounter() uint64 {
	return atomic.LoadUint64(&r.counter)
}

func (r *River) IncrementCounter() {
	atomic.AddUint64(&r.counter, 1)
}

func (r *River) UpdateFeeds() {
	for _, wf := range r.Streams {
		go r.Fetch(wf)
	}
	r.builds += 1
}

func (r *River) Fetch(wf *WebFeed) {
	req, err := http.NewRequest("GET", wf.URL, nil)
	if err != nil {
		errorLog.Printf("error creating request for %q", wf.URL)
		return
	}

	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("From", "https://github.com/edavis/colorado")

	if wf.LastModified != "" {
		req.Header.Add("If-Modified-Since", wf.LastModified)
	}

	if wf.ETag != "" {
		req.Header.Add("If-None-Match", wf.ETag)
	}

	logger.Printf("fetching %q", wf.URL)

	if wf.LastModified != "" || wf.ETag != "" {
		logger.Printf("- sending If-Modified-Since = %v, If-None-Match = %v", wf.LastModified, wf.ETag)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		errorLog.Printf("error requesting %q", wf.URL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		logger.Printf("received HTTP 304 when fetching %q, skipping", wf.URL)
		return
	}

	wf.LastModified = resp.Header.Get("Last-Modified")
	wf.ETag = resp.Header.Get("ETag")

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	fr := FetchResult{URL: wf.URL}

	if err != nil {
		errorLog.Printf("error parsing %s (%v)", wf.URL, err)
	} else {
		fr.Feed = feed
	}

	r.FetchResults <- fr
}

func (r *River) ProcessFeed(result FetchResult) {
	var feed *gofeed.Feed

	if feed = result.Feed; feed == nil {
		return
	}

	feedUrl := result.URL
	newItems := 0

	generateFingerprint := func(url string, item *gofeed.Item) string {
		var guid string

		switch {
		case item.GUID != "":
			guid = item.GUID
		case item.Link != "":
			guid = item.Link
		default:
			guid = uuid.NewV4().String()
		}

		return fmt.Sprintf("%s:%s", url, guid)
	}

	extractBody := func(item *gofeed.Item) string {
		body := ""
		switch {
		case item.Description != "":
			body = item.Description
		case item.Content != "":
			body = item.Content
		}
		return truncateText(makePlainText(body))
	}

	feedUpdate := UpdatedFeed{
		Title:       makePlainText(feed.Title),
		Website:     feed.Link,
		URL:         feedUrl,
		Description: feed.Description,
		LastUpdate:  nowGMT(),
	}

	// Loop through items in reverse so most recent gets higher ID
	for i := len(feed.Items) - 1; i >= 0; i-- {
		item := feed.Items[i]
		fingerprint := generateFingerprint(feedUrl, item)

		if _, seen := r.Seen[fingerprint]; seen {
			continue
		} else {
			logger.Printf("adding %q", fingerprint)
			newItems += 1
			r.Seen[fingerprint] = true
		}

		r.IncrementCounter()
		itemUpdate := UpdatedFeedItem{
			Body:      extractBody(item),
			Id:        fmt.Sprintf("%d", r.GetCounter()),
			Link:      item.Link,
			PermaLink: item.GUID,
			PubDate:   sanitizeDate(item.Published),
			Title:     makePlainText(item.Title),
		}

		feedUpdate.Items = append([]*UpdatedFeedItem{&itemUpdate}, feedUpdate.Items...)
	}

	if len(feedUpdate.Items) > maxItems {
		feedUpdate.Items = feedUpdate.Items[:maxItems]
	}

	if newItems > 0 {
		r.Updates = append([]*UpdatedFeed{&feedUpdate}, r.Updates...)

		if len(r.Updates) > maxFeedUpdates {
			r.Updates = r.Updates[:maxFeedUpdates]
		}

		logger.Printf("added %d new item(s) from %q to river (counter = %d)", newItems, feed.Title, r.GetCounter())
	}
}
