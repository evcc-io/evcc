package plugin

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// responseCache caches block-read payloads keyed by "raddr/pdu_hex" with a
// short TTL.  All aa55udp source blocks sharing the same (host, pdu) pair
// share one UDP exchange per poll cycle — e.g. the four Ppv string registers
// from READ 125 @ 0x891C all hit the cache after the first fetch.
//
// TTL is set to 2 s: long enough to serve all source blocks within one evcc
// poll cycle (which completes in well under 1 s), short enough that the next
// cycle always fetches fresh data.
type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

const (
	responseCacheTTL     = 2 * time.Second
	responseCacheMaxSize = 64 // max number of (host, pdu) pairs to cache
)

var (
	responseCache   = make(map[string]cacheEntry, responseCacheMaxSize)
	responseCacheMu sync.Mutex
)

// fetchBlock returns the response payload for p.pdu, serving from the cache
// if a fresh entry exists, or performing a UDP exchange otherwise.
func (p *AA55UDP) fetchBlock() ([]byte, error) {
	key := p.conn.RemoteAddr().String() + "/" + hex.EncodeToString(p.pdu)

	responseCacheMu.Lock()
	if entry, ok := responseCache[key]; ok && time.Now().Before(entry.expiresAt) {
		payload := entry.payload
		responseCacheMu.Unlock()
		p.log.TRACE.Printf("cache hit for %s pdu=%s", p.conn.RemoteAddr(), hex.EncodeToString(p.pdu))
		return payload, nil
	}
	responseCacheMu.Unlock()

	// Cache miss — send the request.
	packet := append(p.pdu, modbusCRC16(p.pdu)...)

	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}

	payload, err := stripAA55Header(raw)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	responseCacheMu.Lock()
	// Evict expired entries before inserting to keep the map bounded.
	if len(responseCache) >= responseCacheMaxSize {
		now := time.Now()
		for k, v := range responseCache {
			if now.After(v.expiresAt) {
				delete(responseCache, k)
			}
		}
	}
	responseCache[key] = cacheEntry{payload: payload, expiresAt: time.Now().Add(responseCacheTTL)}
	responseCacheMu.Unlock()

	return payload, nil
}

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
