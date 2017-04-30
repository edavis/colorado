package main

import (
	"bytes"
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	utcTimestampFmt   = "Mon, 02 Jan 2006 15:04:05 GMT"
	localTimestampFmt = "Mon, 02 Jan 2006 15:04:05 MST"
)

func NewRiver(feeds []string) *River {
	r := River{
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
	r.Print("starting HTTP server on 127.0.0.1:9000")
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/river", r.serveRiver)
		mux.HandleFunc("/log", r.serveLog)
		log.Fatal(http.ListenAndServe(":9000", mux))
	}()

	r.Print("updating feeds (initial fetch)")
	r.UpdateFeeds()

	ticker := time.NewTicker(15 * time.Minute)
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

	feed_update := UpdatedFeed{
		Title:       feed.Title,
		Website:     feed.Link,
		URL:         feedUrl,
		Description: feed.Description,
		LastUpdate:  nowGMT(),
	}

	for i := len(feed.Items) - 1; i >= 0; i-- {
		item := feed.Items[i]
		fingerprint := generateFingerprint(feedUrl, item)

		if _, ok := r.Seen[fingerprint]; ok {
			continue
		} else {
			r.Printf("adding %q\n", fingerprint)
			newItems += 1
			r.Seen[fingerprint] = true
		}

		r.IncrementCounter()
		item_update := UpdatedFeedItem{
			Title:     item.Title,
			Link:      item.Link,
			PermaLink: item.GUID,
			PubDate:   item.Published,
			Id:        fmt.Sprintf("%d", r.GetCounter()),
		}
		feed_update.Items = append([]*UpdatedFeedItem{&item_update}, feed_update.Items...)
	}

	if newItems > 0 {
		r.Updates = append([]*UpdatedFeed{&feed_update}, r.Updates...)
		r.Printf("added %d new item(s) from %s to river (counter = %d)\n", newItems, feed.Title, r.GetCounter())
	}
}

func main() {
	feeds := []string{
		// Active feeds
		"http://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.atom",
		"http://news.yahoo.com/rss/",
		"http://talkingpointsmemo.com/feed/livewire",
		"http://www.nytimes.com/timeswire/feeds/",
		"https://hnrss.org/ask?description=0",
		"https://hnrss.org/newest?description=0",
		"https://hnrss.org/show?description=0",
		"https://pypi.python.org/pypi?%3Aaction=packages_rss",
		"https://pypi.python.org/pypi?%3Aaction=rss",

		// Golang feeds
		"http://blog.golang.org/feed.atom",
		"http://blog.gopheracademy.com/index.xml",
		"http://confreaks.tv/events/gophercon2014.atom",
		"http://dave.cheney.net/feed",
		"http://elithrar.github.io/atom.xml",
		"http://hnrss.org/newest?q=golang&description=0",
		"http://research.swtch.com/feed.atom",
		"https://groups.google.com/forum/feed/golang-announce/topics/rss.xml?num=15",
		"https://groups.google.com/forum/feed/golang-dev/topics/rss.xml?num=15",
		"https://groups.google.com/forum/feed/golang-nuts/topics/rss.xml?num=15",
	}
	river := NewRiver(feeds)
	fmt.Println("starting up...")
	river.Printf("starting colorado with %d feeds\n", len(feeds))
	river.Run()
}
