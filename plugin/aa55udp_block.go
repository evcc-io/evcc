package plugin

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// cacheEntry holds a cached response payload with its expiration time.
type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

const (
	responseCacheTTL     = 2 * time.Second
	responseCacheMaxSize = 64 // max number of (host, pdu) pairs to cache
)

// responseCacheT caches block-read payloads keyed by "raddr/pdu_hex" with a
// short TTL. All aa55udp source blocks sharing the same (host, pdu) pair
// share one UDP exchange per poll cycle — e.g. the four PV string registers
// from READ 125 @ 0x891C all hit the cache after the first fetch.
//
// TTL is set to 2 s: long enough to serve all source blocks within one evcc
// poll cycle (which completes in well under 1 s), short enough that the next
// cycle always fetches fresh data.
type responseCacheT struct {
	mu   sync.Mutex
	data map[string]cacheEntry
}

func newResponseCacheT() *responseCacheT {
	return &responseCacheT{
		data: make(map[string]cacheEntry, responseCacheMaxSize),
	}
}

// get returns the cached payload if it exists and is fresh, or (nil, false) otherwise.
func (c *responseCacheT) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.data[key]; ok && time.Now().Before(entry.expiresAt) {
		return entry.payload, true
	}
	return nil, false
}

// put inserts a payload into the cache, evicting expired entries if needed.
func (c *responseCacheT) put(key string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.data) >= responseCacheMaxSize {
		now := time.Now()
		for k, v := range c.data {
			if now.After(v.expiresAt) {
				delete(c.data, k)
			}
		}
	}
	c.data[key] = cacheEntry{
		payload:   payload,
		expiresAt: time.Now().Add(responseCacheTTL),
	}
}

var responseCache = newResponseCacheT()

// buildPDUFromHex decodes a hex string (spaces allowed) into exactly 6 bytes,
// representing a complete PDU body including the inverter address byte.
// Used exclusively in block read mode where the caller supplies the full PDU.
func buildPDUFromHex(s string) ([]byte, error) {
	clean := strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: invalid pdu %q: %w", s, err)
	}
	if len(b) != 6 {
		return nil, fmt.Errorf("aa55udp: pdu must be 6 bytes, got %d", len(b))
	}
	return b, nil
}
