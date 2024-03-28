package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const admin = "admin"

type Auth struct {
	settings settings.API
}

func New(settings settings.API) *Auth {
	return &Auth{settings: settings}
}

func (a *Auth) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (a *Auth) getAdminPasswordHash() string {
	if pw, err := a.settings.String(keys.AdminPassword); err == nil {
		return pw
	}
	return ""
}

// RemoveAdminPassword resets the admin password. For recovery mode via cli.
func (a *Auth) RemoveAdminPassword() {
	a.settings.SetString(keys.AdminPassword, "")
	a.settings.SetString(keys.JwtSecret, "")
}

// IsAdminPasswordConfigured checks if the admin password is already set
func (a *Auth) IsAdminPasswordConfigured() bool {
	return a.getAdminPasswordHash() != ""
}

// SetAdminPassword sets the admin password if not already set
func (a *Auth) SetAdminPassword(password string) error {
	if a.getAdminPasswordHash() != "" {
		return errors.New("admin password already set")
	}

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
func (a *Auth) IsAdminPasswordValid(password string) bool {
	adminHash := a.getAdminPasswordHash()
	fmt.Println("adminHash: " + adminHash)
	if adminHash == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(adminHash), []byte(password)) == nil
}

func (a *Auth) generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// getJwtSecret returns the JWT secret from the settings or generates a new one
func (a *Auth) getJwtSecret() ([]byte, error) {
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
func (a *Auth) GenerateJwtToken(lifetime time.Duration) (string, error) {
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
func (a *Auth) ValidateJwtToken(tokenString string) (bool, error) {
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
