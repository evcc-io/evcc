package util

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

// Store is the parameter store database container.
type Store struct {
	name       string
	bucketName []byte
	db         *bolt.DB
}

// Initialize a new persistent key value store in os temp directory
func NewStore(name string) (*Store, error) {
	file := fmt.Sprintf("%s/%s.db", os.TempDir(), name)

	s, err := OpenStore(file, []byte(name))
	if err != nil {
		return nil, err
	}

	return &Store{
		name:       name,
		bucketName: []byte(name),
		db:         s.db,
	}, nil
}

// Open a key-value store. "path" is the full path to the database file, any
// leading directories must have been created already. File is created with
// mode 0640 if needed.
//
// Because of BoltDB restrictions, only one process may open the file at a
// time. Attempts to open the file from another process will fail with a
// timeout error.
func OpenStore(path string, bucketName []byte) (*Store, error) {
	opts := &bolt.Options{
		Timeout: 50 * time.Millisecond,
	}
	if db, err := bolt.Open(path, 0640, opts); err != nil {
		return nil, err
	} else {
		err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(bucketName)
			return err
		})
		if err != nil {
			return nil, err
		} else {
			return &Store{db: db}, nil
		}
	}
}

// Put an entry into the store. The passed value is gob-encoded and stored.
// The key can be an empty string, but the value cannot be nil - if it is,
// Put() returns bad value.
func (s *Store) Put(key string, value interface{}) error {
	if value == nil {
		return errors.New("store: bad value")
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.bucketName).Put([]byte(key), buf.Bytes())
	})
}

// Get an entry from the store. "value" must be a pointer-typed. If the key
// is not present in the store, Get returns key not found.
// The value passed to Get() can be nil, in which case any value read from
// the store is silently discarded.
func (s *Store) Get(key string, value interface{}) error {
	return s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(s.bucketName).Cursor()
		if k, v := c.Seek([]byte(key)); k == nil || string(k) != key {
			return errors.New("store: key not found")
		} else if value == nil {
			return nil
		} else {
			d := gob.NewDecoder(bytes.NewReader(v))
			return d.Decode(value)
		}
	})
}

// Delete the entry with the given key. If no such key is present in the store,
// it returns key not found.
func (s *Store) Delete(key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket(s.bucketName).Cursor()
		if k, _ := c.Seek([]byte(key)); k == nil || string(k) != key {
			return errors.New("store: key not found")
		} else {
			return c.Delete()
		}
	})
}

// CloseStore closes the evcc key-value store file.
func (s *Store) CloseStore() error {
	return s.db.Close()
}
