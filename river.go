package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func NewRiver(name string, feeds []string, updateInterval, title, description string) *River {
	r := River{
		Name:             name,
		Title:            title,
		Description:      description,
		FetchResults:     make(chan FetchResult),
		Seen:             make(map[string]bool),
		whenStartedGMT:   nowGMT(),
		whenStartedLocal: nowLocal(),
		Mutex:            new(sync.Mutex),
		httpClient:       http.DefaultClient,
	}

	duration, err := time.ParseDuration(updateInterval)
	if err != nil {
		fmt.Printf("the duration %q is invalid, using default of 15 minutes\n", updateInterval)
		duration = 15 * time.Minute
	}
	r.UpdateInterval = duration

	for _, feed := range feeds {
		r.Streams = append(r.Streams, feed)
	}
	return &r
}

func (r *River) Run() {
	r.Msg("updating feeds (initial fetch)\n")
	r.UpdateFeeds()

	r.Msg("fetching feeds every %s\n", r.UpdateInterval)
	ticker := time.NewTicker(r.UpdateInterval)

	for {
		select {
		case result := <-r.FetchResults:
			r.ProcessFeed(result)
		case <-ticker.C:
			r.Msg("updating feeds\n")
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
	for _, stream := range r.Streams {
		go r.Fetch(stream)
	}
	r.builds += 1
}

func (r *River) Fetch(url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		r.Msg("error creating request for %q\n", url)
	}

	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("From", "https://github.com/edavis/colorado")
	r.Msg("fetching %q\n", url)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.Msg("error requesting %q\n", url)
	}
	defer resp.Body.Close()

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	fr := FetchResult{URL: url}

	if err != nil {
		r.Msg("error parsing %s (%v)\n", url, err)
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
			r.Msg("adding %q\n", fingerprint)
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

	if newItems > 0 {
		r.Updates = append([]*UpdatedFeed{&feedUpdate}, r.Updates...)
		r.Msg("added %d new item(s) from %q to river (counter = %d)\n", newItems, feed.Title, r.GetCounter())
	}
}

func (r *River) Msg(format string, a ...interface{}) {
	r.Lock()
	now := time.Now().Local().Format(localTimestampFmt)
	event := fmt.Sprintf(format, a...)
	msg := fmt.Sprintf("%s: %s", now, event)
	r.Messages = append(r.Messages, msg)
	if len(r.Messages) > maxEventLog {
		r.Messages = r.Messages[1:]
	}
	r.Unlock()
}
