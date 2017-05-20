package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"net/http"
	"time"
)

type WebFeed struct {
	URL string
}

// River holds the main app logic.
type River struct {
	Name             string
	Title            string
	Description      string
	FetchResults     chan FetchResult
	WebFeedChan      chan *WebFeed
	Streams          []*WebFeed
	UpdateInterval   time.Duration
	builds           uint64
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
		WebFeedChan:      make(chan *WebFeed),
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

	if err = db.Update(createBucket(name)); err != nil {
		errorLog.Println(err)
		logger.Println(err)
	}

	return &r
}

func (r *River) Run() {
	// start the worker and initial feed check
	go r.FetchWorker()
	go r.UpdateFeeds()

	ticker := time.NewTicker(r.UpdateInterval)

	for {
		select {
		case result := <-r.FetchResults:
			r.ProcessFeed(result)
		case <-ticker.C:
			go r.UpdateFeeds()
		}
	}
}

func (r *River) UpdateFeeds() {
	for _, wf := range r.Streams {
		r.WebFeedChan <- wf
	}
	r.builds += 1
}

func (r *River) FetchWorker() {
	for {
		r.Fetch(<-r.WebFeedChan)
	}
}

func (r *River) Fetch(wf *WebFeed) {
	req, err := http.NewRequest("GET", wf.URL, nil)
	if err != nil {
		errorLog.Printf("error creating request for %q (%v)", wf.URL, err)
		return
	}

	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("From", "https://github.com/edavis/colorado")

	err = db.View(getCacheHeaders(r.Name, wf.URL, req))
	if err != nil {
		errorLog.Printf("couldn't set cache headers on request for %q (%v)", wf.URL, err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		errorLog.Printf("error requesting %q (%v)", wf.URL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		logger.Printf("added 0 new item(s) from %q to %s (HTTP 304)", wf.URL, r.Name)
		return
	}

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		errorLog.Printf("error parsing %q (%v)", wf.URL, err)
		return
	}

	// If made it this far, the fetch was a success. Update the cache
	// headers and send a FetchResult to the FetchResults channel.
	err = db.Batch(setCacheHeaders(r.Name, wf.URL, resp))
	if err != nil {
		errorLog.Printf("couldn't update cache headers for %q (%v)", wf.URL, err)
	}

	r.FetchResults <- FetchResult{URL: wf.URL, Feed: feed}
}

func (r *River) ProcessFeed(result FetchResult) {
	feed := result.Feed
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

		var seen bool
		if err := db.Batch(checkFingerprint(r.Name, fingerprint, &seen)); err != nil {
			errorLog.Println(err)
		}

		if seen {
			continue
		} else {
			newItems += 1
		}

		itemUpdate := UpdatedFeedItem{
			Body:      extractBody(item),
			Link:      item.Link,
			PermaLink: item.GUID,
			PubDate:   sanitizeDate(item.Published),
			Title:     makePlainText(item.Title),
		}

		if err := db.Update(assignNextID(r.Name, &itemUpdate)); err != nil {
			errorLog.Printf("error assigning next ID: %v", err)
		}

		feedUpdate.Items = append([]*UpdatedFeedItem{&itemUpdate}, feedUpdate.Items...)
	}

	if len(feedUpdate.Items) > maxItems {
		feedUpdate.Items = feedUpdate.Items[:maxItems]
	}

	if newItems > 0 {
		if err := db.Batch(updateRiver(r.Name, &feedUpdate)); err != nil {
			errorLog.Println(err)
			logger.Println(err)
		}
	}

	logger.Printf("added %d new item(s) from %q to %s", newItems, feedUrl, r.Name)
}
