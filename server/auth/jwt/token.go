package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// New generates an admin user JWT token with the given time to live
func New(subject string, secret []byte, ttl time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func Validate(token, subject string, secret []byte) error {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithSubject(subject))

	return err
}
