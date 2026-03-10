package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/fatih/structs"
)

// Sharder splits data into chunks, omitting unmodified chunks
type Sharder interface {
	ModifiedShards() iter.Seq2[string, any]
	AllShards() iter.Seq2[string, any]
}

type Shard struct {
	Key   string
	Value any
}

type sharderImpl struct {
	prefix string
	struc  any
}

// shared shard cache
var (
	shardCache = make(map[string][32]byte)
	shardMu    sync.Mutex
)

func (s *sharderImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.struc)
}

func (s *sharderImpl) AllShards() iter.Seq2[string, any] {
	return s.shards(false)
}

func (s *sharderImpl) ModifiedShards() iter.Seq2[string, any] {
	return s.shards(true)
}

func (s *sharderImpl) shards(useCache bool) iter.Seq2[string, any] {
	if useCache {
		shardMu.Lock()
		defer shardMu.Unlock()
	}

	return func(yield func(string, any) bool) {
		for _, f := range structs.Fields(s.struc) {
			key := jsonKey(f)
			if useCache && s.skipCachedShard(key, f.Value()) {
				continue
			}
			if !yield(key, f.Value()) {
				break
			}
		}
	}
}

func jsonKey(f *structs.Field) string {
	key := f.Name()
	if t := f.Tag("json"); t != "" {
		if n := strings.Split(t, ",")[0]; n != "" {
			key = n
		}
	}
	return key
}

func (s *sharderImpl) skipCachedShard(key string, value any) bool {
	// Use JSON for stable hashing (fmt.Append includes pointer addresses)
	b, err := json.Marshal(value)
	if err != nil {
		// Fallback to fmt.Append if JSON fails
		b = fmt.Append(nil, value)
	}

	hash := sha256.Sum256(b)
	cacheKey := s.prefix + key

	if cached, ok := shardCache[cacheKey]; ok && hash == cached {
		return true
	}

	shardCache[cacheKey] = hash
	return false
}

var _ api.StructMarshaler = (*sharderImpl)(nil)

func (s *sharderImpl) MarshalStruct() (any, error) {
	return s.struc, nil
}

var _ Sharder = (*sharderImpl)(nil)

// NewSharder creates a Sharder that splits structs into sub-structs for space-efficient socket publishing
// Passing anything else than a struct will panic
func NewSharder(prefix string, struc any) Sharder {
	return &sharderImpl{prefix, struc}
}
