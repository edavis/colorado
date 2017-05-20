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

		// Get the JSON out of boltdb
		b := tx.Bucket([]byte(name))
		obj := b.Get([]byte("river"))

		// Decode the byte slice into a slice of *UpdateFeed and
		// prepend the new update
		if obj != nil {
			json.Unmarshal(obj, &updates)
			updates = append([]*UpdatedFeed{newUpdate}, updates...)
		} else {
			updates = []*UpdatedFeed{newUpdate}
		}

		// Trim the update slice down to size
		if len(updates) > maxFeedUpdates {
			updates = updates[:maxFeedUpdates]
		}

		// Encode the new river object and update bolt with it
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

// checkFingerprint
func checkFingerprint(name, fingerprint string, seen *bool) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		result := b.Get([]byte(fingerprint))
		if result != nil {
			*seen = true
		} else {
			*seen = false
			err := b.Put([]byte(fingerprint), []byte{1})
			if err != nil {
				return err
			}
		}
		return nil
	}
}
