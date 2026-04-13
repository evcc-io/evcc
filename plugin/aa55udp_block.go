package plugin

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// responseCache caches block-read payloads keyed by "raddr/pdu_hex" with a
// short TTL.  This allows multiple aa55udp source blocks that read different
// offsets from the same block PDU (e.g. all four Ppv string registers from
// READ 125 @ 0x891C) to share a single UDP exchange per poll cycle.
type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

var (
	responseCache   = make(map[string]cacheEntry)
	responseCacheMu sync.Mutex
)

// responseCacheTTL must be long enough to cover all sequential source reads
// within one evcc poll cycle (typically < 1 s), but short enough that the
// next cycle fetches fresh data.
const responseCacheTTL = 5 * time.Second

// fetchBlock returns the response payload for p.pdu, serving from the cache
// if a fresh entry exists, or performing a UDP exchange otherwise.
// Multiple AA55UDP instances sharing the same (raddr, pdu) will reuse the
// same cached payload, reducing the number of UDP exchanges per poll cycle.
func (p *AA55UDP) fetchBlock() ([]byte, error) {
	key := p.raddr.String() + "/" + hex.EncodeToString(p.pdu)

	responseCacheMu.Lock()
	if entry, ok := responseCache[key]; ok && time.Now().Before(entry.expiresAt) {
		payload := entry.payload
		responseCacheMu.Unlock()
		p.log.TRACE.Printf("cache hit for %s pdu=%s", p.raddr, hex.EncodeToString(p.pdu))
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
