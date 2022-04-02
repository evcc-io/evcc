package util

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"go.etcd.io/bbolt"
)

// Store is the parameter store database container.
type Store struct {
	Log        *Logger
	dbName     string
	bucketName string
	isOpen     bool
	db         *bbolt.DB
}

// Initialize a new persistent key value store in os user config directory
func NewStore(dbName, bucketName string) *Store {
	if bucketName == "" {
		bucketName = dbName
	}

	s := &Store{
		isOpen:     false,
		dbName:     dbName,
		bucketName: bucketName,
	}

	return s
}

// Open a key-value store. "path" is the full path to the database file, any
// leading directories must have been created already. File is created with
// mode 0640 if needed.
//
// Because of BoltDB restrictions, only one process may open the file at a
// time. Attempts to open the file from another process will fail with a
// timeout error.
func (s *Store) Open() {

	s.Log = NewLogger(fmt.Sprintf("store-%s", s.dbName))

	cachedir, err := os.UserConfigDir()
	if err != nil {
		s.Log.WARN.Printf("cannot determine logdir %s: %v", cachedir, err)
	}

	opts := &bbolt.Options{
		Timeout: 50 * time.Millisecond,
	}

	if db, err := bbolt.Open(fmt.Sprintf("%s/%s.db", cachedir, s.dbName), 0640, opts); err != nil {
		s.Log.WARN.Printf("cannot open %s", fmt.Sprintf("%s/%s.db", cachedir, s.dbName))
	} else {
		s.Log.DEBUG.Printf("%s opened", fmt.Sprintf("%s/%s.db", cachedir, s.dbName))
		err := db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(s.bucketName))
			return err
		})
		if err != nil {
			s.Log.WARN.Printf("open error: %v", err)
		} else {
			s.isOpen = true
			s.db = db
		}
	}
}

// Put an entry into the store. The passed value is gob-encoded and stored.
// The key cannot be an empty string, but the value cannot be nil - if it is,
// Put() is not storing the value
func (s *Store) Put(key string, value interface{}) {
	if key == "" || value == nil || !s.isOpen {
		s.Log.WARN.Printf("put invalid key,value or missing db: %s / %v", key, value)
	} else {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(value); err != nil {
			s.Log.ERROR.Printf("error encoding value %v: %v", value, err)
		}

		err := s.db.Update(func(tx *bbolt.Tx) error {
			return tx.Bucket([]byte(s.bucketName)).Put([]byte(key), buf.Bytes())
		})
		if err != nil {
			s.Log.ERROR.Printf("put error: %v", err)
		}

		s.Log.DEBUG.Printf("put stored key <%s> in bucket <%s> with value <%v>:", key, s.bucketName, value)

	}
}

// Get an entry from the store. "value" must be a pointer-typed. If the key
// is not present in the store, Get is not updating the value
func (s *Store) Get(key string, value interface{}) {
	if key == "" || !s.isOpen {
		s.Log.WARN.Printf("get invalid key or missing db: %s", key)
	} else {
		err := s.db.View(func(tx *bbolt.Tx) error {
			c := tx.Bucket([]byte(s.bucketName)).Cursor()
			if k, v := c.Seek([]byte(key)); k == nil || string(k) != key {
				s.Log.WARN.Printf("get key %s not found", key)
			} else if value != nil {
				d := gob.NewDecoder(bytes.NewReader(v))
				return d.Decode(value)
			}

			return nil
		})

		if err != nil {
			s.Log.ERROR.Printf("get error: %v", err)
		}
	}
}

// Delete the entry with the given key. If no such key is present in the store,
// it returns key not found.
func (s *Store) Delete(key string) {
	if key == "" || !s.isOpen {
		s.Log.WARN.Printf("delete invalid key or missing db: %s", key)
	} else {
		err := s.db.Update(func(tx *bbolt.Tx) error {
			c := tx.Bucket([]byte(s.bucketName)).Cursor()
			if k, _ := c.Seek([]byte(key)); k == nil || string(k) != key {
				s.Log.WARN.Printf("delete key %s not found", key)
			} else {
				return c.Delete()
			}
			return nil
		})

		if err != nil {
			s.Log.ERROR.Printf("delete error: %v", err)
		}
	}
}

// Closes the evcc key-value store file.
func (s *Store) Close() {
	if !s.isOpen {
		s.Log.WARN.Print("close missing db")
	} else {
		err := s.db.Close()
		if err != nil {
			s.Log.ERROR.Printf("delete error: %v", err)
		}
	}
}
