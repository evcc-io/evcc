package uilock

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	uilockCookieName = "uilock"
	uilockSubject    = "uilock"
	// MaskedPin is the placeholder sent to the UI when a PIN is configured (compare Influx etc.).
	MaskedPin = "***"
)

// Manager holds UI lock state and validates unlock tokens.
type Manager struct {
	settings settings.API
}

// NewManager creates a Manager using the default settings store.
func NewManager() *Manager {
	return &Manager{settings: new(settings.Settings)}
}

// NewManagerWithSettings allows tests to inject a mock settings API.
func NewManagerWithSettings(s settings.API) *Manager {
	return &Manager{settings: s}
}

// PinConfigured returns whether a PIN hash exists.
func (m *Manager) PinConfigured() bool {
	h, err := m.settings.String(keys.UiLockPin)
	return err == nil && h != ""
}

// SetPin stores a bcrypt hash of the PIN.
func (m *Manager) SetPin(pin string) error {
	if pin == "" {
		m.settings.SetString(keys.UiLockPin, "")
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	m.settings.SetString(keys.UiLockPin, string(hash))
	return nil
}

// IsPinValid checks the PIN against the stored hash.
func (m *Manager) IsPinValid(pin string) bool {
	h, err := m.settings.String(keys.UiLockPin)
	if err != nil || h == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(pin)) == nil
}

// Published builds the value broadcast to the UI.
func (m *Manager) Published(conf globalconfig.UILock) globalconfig.UILockPublished {
	pinField := ""
	if m.PinConfigured() {
		pinField = MaskedPin
	}
	return globalconfig.UILockPublished{
		Enabled:        conf.Enabled,
		Timeout:        conf.Timeout.Seconds(),
		IPs:            conf.IPs,
		TrustedProxies: conf.TrustedProxies,
		Pin:            pinField,
		PinConfigured:  m.PinConfigured(),
	}
}

// Applies returns whether the UI lock applies to this request (PIN required but not yet unlocked session).
func (m *Manager) Applies(r *http.Request, conf globalconfig.UILock) bool {
	if !conf.Enabled || !m.PinConfigured() {
		return false
	}
	nets := ParseCIDRList(conf.TrustedProxies)
	clientIP := EffectiveClientIP(r, nets)
	if clientIP == nil {
		return false
	}
	return ipMatchesList(clientIP, conf.IPs)
}

// ipMatchesList reports whether ip matches any entry (string form, IPv4/IPv6).
func ipMatchesList(ip net.IP, list []string) bool {
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if other := net.ParseIP(s); other != nil && ip.Equal(other) {
			return true
		}
	}
	return false
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) jwtSecret() ([]byte, error) {
	s, err := m.settings.String(keys.JwtSecret)
	if err != nil || s == "" {
		key, err := randomHex(32)
		if err != nil {
			return nil, err
		}
		m.settings.SetString(keys.JwtSecret, key)
		return []byte(key), nil
	}
	return []byte(s), nil
}

// IssueUnlockCookie sets the uilock cookie with a JWT valid for the given idle duration.
func (m *Manager) IssueUnlockCookie(w http.ResponseWriter, conf globalconfig.UILock) error {
	secret, err := m.jwtSecret()
	if err != nil {
		return err
	}
	if conf.Timeout <= 0 {
		conf.Timeout = globalconfig.DefaultUILockTimeout
	}
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		Subject:   uilockSubject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(conf.Timeout)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(secret)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     uilockCookieName,
		Value:    s,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  now.Add(conf.Timeout),
	})
	return nil
}

// HasValidUnlockToken reports whether the uilock cookie is present and still valid (not expired).
func (m *Manager) HasValidUnlockToken(r *http.Request) bool {
	c, err := r.Cookie(uilockCookieName)
	if err != nil || c == nil || c.Value == "" {
		return false
	}
	secret, err := m.jwtSecret()
	if err != nil {
		return false
	}
	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(c.Value, &claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil || !token.Valid || claims.Subject != uilockSubject {
		return false
	}
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return false
	}
	return true
}

// RefreshUnlockCookieIfValid re-issues the cookie when the current token is valid (sliding idle timeout).
func (m *Manager) RefreshUnlockCookieIfValid(w http.ResponseWriter, r *http.Request, conf globalconfig.UILock) bool {
	if !m.HasValidUnlockToken(r) {
		return false
	}
	return m.IssueUnlockCookie(w, conf) == nil
}

// ClearUnlockCookie clears the uilock cookie.
func ClearUnlockCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     uilockCookieName,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
