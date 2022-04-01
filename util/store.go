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

// SavingsStore is the parameter store container.
type Store struct {
	name       string
	bucketName []byte
	db         *bolt.DB
}

var (
	// ErrNotFound is returned when the key supplied to a Get or Delete
	// method does not exist in the database.
	ErrNotFound = errors.New("skv: key not found")

	// ErrBadValue is returned when the value supplied to the Put method
	// is nil.
	ErrBadValue = errors.New("skv: bad value")
)

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

func (s *Store) Put(key string, value interface{}) error {
	if value == nil {
		return ErrBadValue
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.bucketName).Put([]byte(key), buf.Bytes())
	})
}

func (s *Store) Get(key string, value interface{}) error {
	return s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(s.bucketName).Cursor()
		if k, v := c.Seek([]byte(key)); k == nil || string(k) != key {
			return ErrNotFound
		} else if value == nil {
			return nil
		} else {
			d := gob.NewDecoder(bytes.NewReader(v))
			return d.Decode(value)
		}
	})
}

// CloseStore closes the evcc key-value store file.
func (s *Store) CloseStore() error {
	return s.db.Close()
}
