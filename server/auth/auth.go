package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/auth/api"
	"github.com/evcc-io/evcc/server/auth/jwt"
	"github.com/evcc-io/evcc/server/auth/keys"
	"github.com/evcc-io/evcc/server/db/settings"

	// "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const admin = "admin"

// Possible authentication modes
type AuthMode int

const (
	Enabled  AuthMode = iota // normal operation
	Disabled                 // auth checks are skipped (free for all)
	Locked                   // auth features are blocked (demo mode)
)

// Auth is the Auth api
type Auth interface {
	RemoveAdminPassword()
	SetAdminPassword(string) error
	IsAdminPasswordValid(string) bool
	GenerateJwtToken(time.Duration) (string, error)
	ValidateToken(string) error
	IsAdminPasswordConfigured() bool
	SetAuthMode(AuthMode)
	GetAuthMode() AuthMode
}

type auth struct {
	settings settings.API
	authMode AuthMode
}

func New() Auth {
	return &auth{settings: new(settings.Settings), authMode: Enabled}
}

func NewMock(settings settings.API) Auth {
	return &auth{settings: settings, authMode: Enabled}
}

func (a *auth) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (a *auth) getAdminPasswordHash() string {
	if pw, err := a.settings.String(keys.AdminPassword); err == nil {
		return pw
	}
	return ""
}

// RemoveAdminPassword resets the admin password. For recovery mode via cli.
func (a *auth) RemoveAdminPassword() {
	a.settings.SetString(keys.AdminPassword, "")
	a.settings.SetString(keys.JwtSecret, "")
}

// IsAdminPasswordConfigured checks if the admin password is already set
func (a *auth) IsAdminPasswordConfigured() bool {
	return a.getAdminPasswordHash() != ""
}

// SetAdminPassword sets the admin password if not already set
func (a *auth) SetAdminPassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}

	hashed, err := a.hashPassword(password)
	if err != nil {
		return err
	}

	a.settings.SetString(keys.AdminPassword, hashed)
	return nil
}

// IsAdminPasswordValid checks if the given password matches the admin password
func (a *auth) IsAdminPasswordValid(password string) bool {
	adminHash := a.getAdminPasswordHash()
	if adminHash == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(adminHash), []byte(password)) == nil
}

func (a *auth) generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// tokenSecret returns the token secret from the settings or generates a new one
func (a *auth) tokenSecret() ([]byte, error) {
	jwtSecret, err := a.settings.String(keys.JwtSecret)

	// generate new secret if it doesn't exist yet -> new installation
	if err != nil || jwtSecret == "" {
		jwtSecret, err = a.generateRandomKey(32)
		if err != nil {
			return nil, err
		}
		a.settings.SetString(keys.JwtSecret, jwtSecret)
	}

	return []byte(jwtSecret), nil
}

// GenerateJwtToken generates an admin user JWT token with the given time to live
func (a *auth) GenerateJwtToken(ttl time.Duration) (string, error) {
	secret, err := a.tokenSecret()
	if err != nil {
		return "", err
	}

	return jwt.New(admin, secret, ttl)
}

// ValidateToken validates the given JWT token
func (a *auth) ValidateToken(token string) error {
	secret, err := a.tokenSecret()
	if err != nil {
		return err
	}

	if strings.HasPrefix(token, api.Prefix) {
		return api.Validate(token, secret)
	}

	return jwt.Validate(token, admin, secret)
}

func (a *auth) SetAuthMode(authMode AuthMode) {
	a.authMode = authMode
}

func (a *auth) GetAuthMode() AuthMode {
	return a.authMode
}
