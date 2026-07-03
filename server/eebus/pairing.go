package eebus

import (
	"encoding/hex"
	"errors"
	"sync"

	shipapi "github.com/enbility/ship-go/api"
	"github.com/evcc-io/evcc/server/db/settings"
)

const (
	ringBufferKey    = "eebus.pairing.ringbuffer"
	trustedDeviceKey = "eebus.pairing.trusted"
)

// trustedDevices returns the persisted identities of devices paired via the
// SHIP Pairing Service
func trustedDevices() ([]shipapi.ServiceIdentity, error) {
	var identities []shipapi.ServiceIdentity
	err := settings.Json(trustedDeviceKey, &identities)
	if errors.Is(err, settings.ErrNotFound) {
		err = nil
	}
	return identities, err
}

// storeTrustedDevices persists the identities of devices paired via the
// SHIP Pairing Service
func storeTrustedDevices(identities []shipapi.ServiceIdentity) error {
	return settings.SetJson(trustedDeviceKey, identities)
}

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

// ringBuffer persists the SHIP pairing replay-protection digests via settings.
type ringBuffer struct {
	mu sync.Mutex
}

type ringBufferState struct {
	Entries   []shipapi.DigestEntry `json:"entries"`
	NextIndex int                   `json:"nextIndex"`
}

func (r *ringBuffer) LoadRingBuffer() ([]shipapi.DigestEntry, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var s ringBufferState
	if err := settings.Json(ringBufferKey, &s); err != nil {
		// no data yet is not an error
		if errors.Is(err, settings.ErrNotFound) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return s.Entries, s.NextIndex, nil
}

func (r *ringBuffer) SaveRingBuffer(entries []shipapi.DigestEntry, nextIndex int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return settings.SetJson(ringBufferKey, ringBufferState{Entries: entries, NextIndex: nextIndex})
}
