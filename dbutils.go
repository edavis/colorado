package main

import (
	"encoding/json"
	"github.com/boltdb/bolt"
)

// createBucket creates the bucket if it does not exist.
func createBucket(name string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
			return err
		}
		return nil
	}
}

// updateRiver prepends the new feed update to the stored JSON.
func updateRiver(name string, newUpdate *UpdatedFeed) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		var updates []*UpdatedFeed

		// 1) Get the JSON out of bolt
		b := tx.Bucket([]byte(name))
		obj := b.Get([]byte("river"))

		// 2) Decode the byte slice into a slice of *UpdateFeed and
		// prepend the new update
		if obj != nil {
			json.Unmarshal(obj, &updates)
			updates = append([]*UpdatedFeed{newUpdate}, updates...)
		} else {
			updates = []*UpdatedFeed{newUpdate}
		}

		// 3) Encode the new river object and update bolt with it
		updatedRiver, err := json.Marshal(updates)
		err = b.Put([]byte("river"), updatedRiver)
		if err != nil {
			return err
		}

		return nil
	}
}

// getRiver places the slice of *UpdateFeeds onto the RiverJS struct.
func getRiver(name string, js *RiverJS) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		var updates []*UpdatedFeed
		b := tx.Bucket([]byte(name))
		raw := b.Get([]byte("river"))
		json.Unmarshal(raw, &updates)
		js.UpdatedFeeds.UpdatedFeed = updates
		return nil
	}
}
