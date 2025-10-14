package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/structs"
)

// Sharder splits data into chunks, omitting unmodified chunks
type Sharder interface {
	Shards() []Shard
}

type Shard struct {
	Key   string
	Value any
}

type sharderImpl struct {
	cache map[string][32]byte
	struc any
}

func (s *sharderImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.struc)
}

func (s *sharderImpl) Shards() []Shard {
	ff := structs.Fields(s.struc)
	res := make([]Shard, 0, len(ff))

	for _, f := range ff {
		key := f.Name()
		if t := f.Tag("json"); t != "" {
			if n := strings.Split(t, ",")[0]; n != "" {
				key = n
			}
		}

		// Use JSON for stable hashing (fmt.Append includes pointer addresses)
		b, err := json.Marshal(f.Value())
		if err != nil {
			// Fallback to fmt.Append if JSON fails
			b = fmt.Append(nil, f.Value())
		}

		hash := sha256.Sum256(b)
		if cached, ok := s.cache[key]; ok && hash == cached {
			continue
		}

		s.cache[key] = hash

		res = append(res, Shard{
			Key:   key,
			Value: f.Value(),
		})
	}

	return res
}

var _ Sharder = (*sharderImpl)(nil)

// NewSharder creates a Sharder that splits structs into sub-structs for space-efficient socket publishing
// Passing anything else than a struct will panic
func NewSharder(cache map[string][32]byte, struc any) Sharder {
	return &sharderImpl{cache, struc}
}
