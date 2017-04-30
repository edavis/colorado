package main

import (
	"bytes"
	"github.com/mmcdole/gofeed"
	"log"
)

// RiverJS is the root JSON returned by /river.
type RiverJS struct {
	Metadata     map[string]string `json:"metadata"`
	UpdatedFeeds UpdatedFeeds      `json:"updatedFeeds"`
}

// UpdatedFeeds is the container that stores the updates.
type UpdatedFeeds struct {
	UpdatedFeed []*UpdatedFeed `json:"updatedFeed"`
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

// River holds the main app logic.
type River struct {
	FetchResults     chan FetchResult
	Streams          []string
	Updates          []*UpdatedFeed
	Seen             map[string]bool
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