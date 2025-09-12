package machine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/samber/lo"
)

var (
	id           string
	getMachineID = machineid.ID
)

// CustomID sets the machine id to a custom value
func CustomID(cid string) error {
	if id != "" {
		panic("machine id already generated")
	}

	cid = strings.TrimSpace(cid)
	if l := len(cid); l != 32 && l != 64 {
		return fmt.Errorf("expected 32 or 64 characters machine id, got %d", l)
	}

	id = cid

	return nil
}

// RandomID creates a random id
func RandomID() string {
	rnd := lo.RandomString(512, lo.LettersCharset)
	mac := hmac.New(sha256.New, []byte(rnd))
	return hex.EncodeToString(mac.Sum(nil))
}

// ID returns the platform specific machine id of the current host OS.
// If ID cannot be generated, a random one from settings will be used (generated on demand)
func ID() string {
	if id == "" {
		var err error
		if id, err = getMachineID(); err == nil && id != "" {
			return id
		}

		// no machine id found, use is from settings
		return getOrCreateIDFromSettings()
	}

	return id
}

// getOrCreateIDFromSettings return instance id from settings if exists, otherwise creates and stores a new one
func getOrCreateIDFromSettings() string {
	if id, err := settings.String(keys.Plant); err == nil && id != "" {
		return id
	}

	id := RandomID()
	settings.SetString(keys.Plant, id)

	return id
}

// ProtectedID returns a hashed version of the machine id
// using a fixed, application-specific key.
func ProtectedID(key string) string {
	return protect(key, ID())
}

// protect calculates HMAC-SHA256 of the id, keyed by key and returns a hex-encoded string
func protect(key, id string) string {
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}
