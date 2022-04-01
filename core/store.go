package core

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
	name string
	file string
	db   *bolt.DB
}

var (
	// ErrNotFound is returned when the key supplied to a Get or Delete
	// method does not exist in the database.
	ErrNotFound = errors.New("store: key not found")

	// ErrBadValue is returned when the value supplied to the Put method
	// is nil.
	ErrBadValue = errors.New("store: bad value")

	bucketName = []byte("evcc")
)

func NewStore(name string) (*Store, error) {
	file := fmt.Sprintf("%s/%s.db", os.TempDir(), name)

	s, err := OpenStore(file)
	if err != nil {
		return nil, err
	}

	return &Store{
		name: name,
		file: file,
		db:   s.db,
	}, nil
}

func OpenStore(path string) (*Store, error) {
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
		return tx.Bucket(bucketName).Put([]byte(key), buf.Bytes())
	})
}

func (s *Store) Get(key string, value interface{}) error {
	return s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketName).Cursor()
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
