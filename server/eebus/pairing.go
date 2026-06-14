package eebus

import (
	"encoding/hex"
	"sync"

	shipapi "github.com/enbility/ship-go/api"
)

// pairing builds the SHIP Pairing Service config (listener mode) and a ring
// buffer from the hex secret. An empty secret disables SHIP pairing.
func pairing(secret string) (*shipapi.PairingConfig, shipapi.RingBufferPersistence, error) {
	if secret == "" {
		return nil, nil, nil
	}

	b, err := hex.DecodeString(secret)
	if err != nil {
		return nil, nil, err
	}

	return shipapi.NewPairingConfig(shipapi.PairingModeListener, shipapi.PairingSecret(b)), new(ringBuffer), nil
}

// ringBuffer is an in-memory RingBufferPersistence for SHIP pairing replay
// protection. TODO: persist across restarts so it survives a restart.
type ringBuffer struct {
	mu        sync.Mutex
	entries   []shipapi.DigestEntry
	nextIndex int
}

func (r *ringBuffer) LoadRingBuffer() ([]shipapi.DigestEntry, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entries, r.nextIndex, nil
}

func (r *ringBuffer) SaveRingBuffer(entries []shipapi.DigestEntry, nextIndex int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = entries
	r.nextIndex = nextIndex
	return nil
}
