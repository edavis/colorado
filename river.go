package main

import (
	"bytes"
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"log"
	"sync/atomic"
	"time"
)

func NewRiver(name string, feeds []string) *River {
	r := River{
		Name:             name,
		FetchResults:     make(chan FetchResult),
		Seen:             make(map[string]bool),
		whenStartedGMT:   nowGMT(),
		whenStartedLocal: nowLocal(),
		buffer:           new(bytes.Buffer),
	}
	r.Logger = log.New(r.buffer, "", log.LstdFlags|log.Lmicroseconds)
	for _, feed := range feeds {
		r.Streams = append(r.Streams, feed)
	}
	return &r
}

func (r *River) Run() {
	r.Print("updating feeds (initial fetch)")
	r.UpdateFeeds()

	// ticker := time.NewTicker(15 * time.Minute)
	ticker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case result := <-r.FetchResults:
			r.ProcessFeed(result)
		case <-ticker.C:
			r.Print("updating feeds")
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
	parser := gofeed.NewParser()
	r.Printf("fetching %s\n", url)
	feed, err := parser.ParseURL(url)
	fr := FetchResult{URL: url}
	if err != nil {
		r.Printf("error parsing %s (%v)\n", url, err)
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

	feedUpdate := UpdatedFeed{
		Title:       feed.Title,
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
			r.Printf("adding %q\n", fingerprint)
			newItems += 1
			r.Seen[fingerprint] = true
		}

		r.IncrementCounter()
		itemUpdate := UpdatedFeedItem{
			Title:     item.Title,
			Link:      item.Link,
			PermaLink: item.GUID,
			PubDate:   sanitizeDate(item.Published),
			Id:        fmt.Sprintf("%d", r.GetCounter()),
		}
		feedUpdate.Items = append([]*UpdatedFeedItem{&itemUpdate}, feedUpdate.Items...)
	}

	if newItems > 0 {
		r.Updates = append([]*UpdatedFeed{&feedUpdate}, r.Updates...)
		r.Printf("added %d new item(s) from %s to river (counter = %d)\n", newItems, feed.Title, r.GetCounter())
	}
}
