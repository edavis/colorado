package main

import (
	"bytes"
	"github.com/mmcdole/gofeed"
	"log"
	"time"
)

// RiverJS is the root JSON returned by /river.
type RiverJS struct {
	Metadata     map[string]string `json:"metadata"`
	UpdatedFeeds struct {
		UpdatedFeed []*UpdatedFeed `json:"updatedFeed"`
	} `json:"updatedFeeds"`
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

type RiverContainer struct {
	Rivers map[string]*River
}

// River holds the main app logic.
type River struct {
	Name             string
	FetchResults     chan FetchResult
	Streams          []string
	Updates          []*UpdatedFeed
	Seen             map[string]bool
	UpdateInterval   time.Duration
	builds           uint64
	counter          uint64 // Item id counter
	whenStartedGMT   string // Track startup times
	whenStartedLocal string
	buffer           *bytes.Buffer // Store logs here
	*log.Logger
}

// FetchResult holds the URL of the feed and its parsed representation.
type FetchResult struct {
	URL  string
	Feed *gofeed.Feed
}
