package database

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "github.com/lib/pq"
	"go.etcd.io/bbolt"
)

var (
	boltDB   *bbolt.DB
	initOnce sync.Once // Ensures BoltDB is only initialized once
)

type User struct {
	Address string   `json:"address"`
	Sites   []string `json:"sites"`
}

// Site represents site data.
type Site struct {
	Name        string `json:"name"`
	IPFSHash    string `json:"ipfs_hash"` // production
	PreviewData string `json:"preview_data"`
}

func BoltInit(path string) error {
	var err error
	initOnce.Do(func() { // Ensures initialization happens only once
		log.Println("Attempting to open database connection...")

		boltDB, err = bbolt.Open(path, 0666, nil)
		if err != nil {
			log.Fatalf("Error opening database: %v", err)
			return
		}
		err = boltDB.Update(func(tx *bbolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists([]byte("Users")); err != nil {
				return fmt.Errorf("create Users bucket: %w", err)
			}
			if _, err := tx.CreateBucketIfNotExists([]byte("Sites")); err != nil {
				return fmt.Errorf("create Sites bucket: %w", err)
			}
			return nil
		})

		if err != nil {
			log.Fatalf("Error creating buckets: %v", err)
			boltDB.Close()
			boltDB = nil
		}
	})

	return err
}

func Close() {
	if err := boltDB.Close(); err != nil {
		log.Println("Error closing database:", err)
	}
}

func Put(bucket, key string, value interface{}) error {
	if boltDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return boltDB.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(key), data)
	})
}

func Get(bucket, key string, out interface{}) error {
	if boltDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return boltDB.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		data := bkt.Get([]byte(key))
		if data == nil {
			log.Println("key not found")
			return fmt.Errorf("key %q not found", key)
		}
		log.Println("key found")
		return json.Unmarshal(data, out)
	})
}

// Delete removes a key from the given bucket.
func Delete(bucket, key string) error {
	if boltDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return boltDB.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		return bkt.Delete([]byte(key))
	})
}
