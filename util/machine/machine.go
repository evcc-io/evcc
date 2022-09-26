package machine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"

	"github.com/denisbrodbeck/machineid"
)

var id string

// CustomID sets the machine id to a custom value
func CustomID(cid string) error {
	if id != "" {
		panic("machine id already generated")
	}

	cid = strings.TrimSpace(cid)
	if l := len(cid); l != 32 {
		return fmt.Errorf("expected 32 characters machine id, got %d", l)
	}

	id = cid

	return nil
}

// ID returns the platform specific machine id of the current host OS.
// If ID cannot be generated, a random value is suggested.
func ID() (string, error) {
	if id == "" {
		var err error
		if id, err = machineid.ID(); err != nil {
			rnd := util.RandomString(512)
			mac := hmac.New(sha256.New, []byte(rnd))
			rid := hex.EncodeToString(mac.Sum(nil))

			return "", fmt.Errorf("could not get %w; for manual configuration use plant: %s", err, rid)
		}
	}

	return id, nil
}

// ProtectedID returns a hashed version of the machine id
// using a fixed, application-specific key.
func ProtectedID(key string) (string, error) {
	id, err := machineid.ID()
	if err != nil {
		return id, err
	}

	return protect(key, id), nil
}

// protect calculates HMAC-SHA256 of the id, keyed by key and returns a hex-encoded string
func protect(key, id string) string {
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}
