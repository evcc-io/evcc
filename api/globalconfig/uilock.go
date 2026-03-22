package globalconfig

import (
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	// DefaultUILockTimeout is the default idle timeout before the UI lock asks for the PIN again.
	DefaultUILockTimeout = 5 * time.Minute
)

// UILock configures an optional PIN lock for the web UI for selected client IPs.
type UILock struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	Timeout        time.Duration `json:"-" yaml:"timeout"`
	IPs            []string      `json:"ips" yaml:"ips"`
	TrustedProxies []string      `json:"trustedProxies" yaml:"trustedProxies"`
	// Pin is only read from YAML for one-time bootstrap; it is not persisted in JSON settings.
	Pin string `json:"-" yaml:"pin,omitempty"`
}

// DefaultUILock returns defaults: disabled, 5m timeout, localhost only.
func DefaultUILock() UILock {
	return UILock{
		Enabled: false,
		Timeout: DefaultUILockTimeout,
		IPs:     []string{"127.0.0.1", "::1"},
	}
}

// MarshalJSON encodes timeout as seconds for the UI.
func (u UILock) MarshalJSON() ([]byte, error) {
	type out struct {
		Enabled        bool     `json:"enabled"`
		Timeout        float64  `json:"timeout"`
		IPs            []string `json:"ips"`
		TrustedProxies []string `json:"trustedProxies"`
		Pin            string   `json:"pin,omitempty"`
	}
	return json.Marshal(out{
		Enabled:        u.Enabled,
		Timeout:        u.Timeout.Seconds(),
		IPs:            u.IPs,
		TrustedProxies: u.TrustedProxies,
		Pin:            u.Pin,
	})
}

// UnmarshalJSON decodes timeout from a JSON number (seconds).
func (u *UILock) UnmarshalJSON(data []byte) error {
	var aux struct {
		Enabled        bool     `json:"enabled"`
		Timeout        float64  `json:"timeout"`
		IPs            []string `json:"ips"`
		TrustedProxies []string `json:"trustedProxies"`
		Pin            string   `json:"pin"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.Enabled = aux.Enabled
	u.Timeout = time.Duration(aux.Timeout * float64(time.Second))
	u.IPs = aux.IPs
	u.TrustedProxies = aux.TrustedProxies
	u.Pin = aux.Pin
	return nil
}

// UILockPublished is the shape broadcast to the UI (includes masked pin field).
type UILockPublished struct {
	Enabled        bool     `json:"enabled"`
	Timeout        float64  `json:"timeout"`
	IPs            []string `json:"ips"`
	TrustedProxies []string `json:"trustedProxies"`
	Pin            string   `json:"pin"`
	PinConfigured  bool     `json:"pinConfigured"`
}

var _ api.Redactor = UILockPublished{}

// Redacted implements api.Redactor.
func (u UILockPublished) Redacted() any {
	return u
}
