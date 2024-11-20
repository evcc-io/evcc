package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const admin = "admin"

// Auth is the Auth api
type Auth interface {
	RemoveAdminPassword()
	SetAdminPassword(string) error
	IsAdminPasswordValid(string) bool
	GenerateJwtToken(time.Duration) (string, error)
	ValidateJwtToken(string) (bool, error)
	IsAdminPasswordConfigured() bool
	Disable()
	Disabled() bool
}

type auth struct {
	settings settings.API
	disabled bool
}

func New() Auth {
	return &auth{settings: new(settings.Settings), disabled: false}
}

func NewMock(settings settings.API) Auth {
	return &auth{settings: settings, disabled: false}
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

// getJwtSecret returns the JWT secret from the settings or generates a new one
func (a *auth) getJwtSecret() ([]byte, error) {
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

// GenerateJwtToken generates an admin user JWT token with the given lifetime
func (a *auth) GenerateJwtToken(lifetime time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   admin,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifetime)),
	}

	if jwtSecret, err := a.getJwtSecret(); err != nil {
		return "", err
	} else {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		return token.SignedString(jwtSecret)
	}
}

// ValidateJwtToken validates the given JWT token
func (a *auth) ValidateJwtToken(tokenString string) (bool, error) {
	jwtSecret, err := a.getJwtSecret()
	if err != nil {
		return false, err
	}

	// read token
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, jwt.WithSubject(admin)); err != nil {
		return false, err
	}

	return true, nil
}

func (a *auth) Disable() {
	a.disabled = true
}

func (a *auth) Disabled() bool {
	return a.disabled
}
