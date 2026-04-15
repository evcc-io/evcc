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

const responseCacheTTL = 2 * time.Second

// blockCache is the package-level response cache shared across all AA55UDP
// instances.  Sharing at package level ensures that multiple source blocks
// for the same (host, pdu) — e.g. the four Ppv string registers all using
// READ 125 @ 0x891C — truly share one UDP exchange per poll cycle.
var blockCache = newResponseCacheT()

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
		data: make(map[string]cacheEntry),
	}
}

// get returns the cached payload if it exists and is fresh, or (nil, false) otherwise.
// Expired entries are deleted on access.
func (c *responseCacheT) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, false
	}
	return entry.payload, true
}

// put inserts or overwrites a payload in the cache.
func (c *responseCacheT) put(key string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		payload:   payload,
		expiresAt: time.Now().Add(responseCacheTTL),
	}
}

// pduFromHex decodes a hex string (spaces allowed) into a PDU of the specified length.
func pduFromHex(s string, wantLen int, context string) ([]byte, error) {
	clean := strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: invalid %s %q: %w", context, s, err)
	}
	if len(b) != wantLen {
		return nil, fmt.Errorf("aa55udp: %s must be %d bytes, got %d", context, wantLen, len(b))
	}
	return b, nil
}

// buildPDUFromHex decodes a hex string (spaces allowed) into exactly 6 bytes,
// representing a complete PDU body including the inverter address byte.
func buildPDUFromHex(s string) ([]byte, error) {
	return pduFromHex(s, 6, "pdu")
}
