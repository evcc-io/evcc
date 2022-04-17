package store

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"go.etcd.io/bbolt"
)

type Store interface {
	Get(string, any) error
	Put(string, any) error
}

// func New(prefix string) Store {
// 	return &storeImpl{prefix: prefix}
// }

// type storeImpl struct {
// 	prefix string
// }

// func (s *storeImplImpl) Get(string, interface{}) error {
// 	return api.ErrNotAvailable
// }

// func (s *storeImplImpl) Put(string, interface{}) error {
// 	return api.ErrNotAvailable
// }

var (
	FileName string
	db       *bbolt.DB
)

func init() {
	dir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	FileName = fmt.Sprintf("%s/%s.db", dir, "evcc")
}

func Create() (err error) {
	db, err = Open(FileName)
	return err
}

// Open a key-value store file. If the file does not exist, it is created.
func Open(file string) (*bbolt.DB, error) {
	opts := &bbolt.Options{
		Timeout: 100 * time.Millisecond,
	}

	db, err := bbolt.Open(file, 0600, opts)
	if err != nil {
		return nil, err
	}

	return db, err
}

// Close closes the store file
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// Store is the parameter store database container.
type storeImpl struct {
	db     *bbolt.DB
	bucket string
}

// Initialize a new persistent key value store in os user config directory
func New(bucket string) Store {
	if db == nil {
		bucket = ""
	} else {
		if err := db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			return err
		}); err != nil {
			bucket = ""
		}
	}

	s := &storeImpl{
		db:     db,
		bucket: bucket,
	}

	return s
}

// Put an entry into the store. The passed value is gob-encoded and stored.
// The key cannot be an empty string, but the value cannot be nil - if it is,
// Put() is not storing the value
func (s *storeImpl) Put(key string, value interface{}) error {
	// TODO fail silently
	if s.bucket == "" {
		return nil
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(s.bucket)).Put([]byte(key), buf.Bytes())
	})
}

// Get an entry from the store. "value" must be a pointer-typed. If the key
// is not present in the store, Get is not updating the value
func (s *storeImpl) Get(key string, value interface{}) error {
	// TODO fail silently
	if s.bucket == "" {
		return nil
	}

	return s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte(s.bucket)).Cursor()

		k, v := c.Seek([]byte(key))
		if k != nil {
			d := gob.NewDecoder(bytes.NewReader(v))
			return d.Decode(value)
		}

		return nil
	})
}

// // Delete the entry with the given key. If no such key is present in the store,
// // it returns key not found.
// func (s *storeImpl) Delete(key string) {
// 	if key == "" || !s.isOpen {
// 	} else {
// 		err := s.db.Update(func(tx *bbolt.Tx) error {
// 			c := tx.Bucket([]byte(s.bucket)).Cursor()
// 			if k, _ := c.Seek([]byte(key)); k == nil || string(k) != key {
// 			} else {
// 				return c.Delete()
// 			}
// 			return nil
// 		})

// 		if err != nil {
// 		}
// 	}
// }
