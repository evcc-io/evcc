package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/andig/evcc/soc/server/auth/sponsor"
	"github.com/andig/evcc/util"
	"github.com/dgrijalva/jwt-go"
)

var (
	Issuer      = "evcc.io"
	TokenExpiry = 365 // days
	tokenSecret = util.Getenv("JWT_TOKEN_SECRET")
)

var (
	mux        sync.Mutex
	updated    time.Time
	authorized = make(map[string]bool)
)

const updateInterval = 5 * time.Minute

type Claims struct {
	Username      string `json:"username"`
	SponsorExempt bool   `json:"spe"`
	jwt.StandardClaims
}

func keyFunc(token *jwt.Token) (interface{}, error) {
	return []byte(tokenSecret), nil
}

func AuthorizedToken(claims Claims) (string, error) {
	claims.Issuer = Issuer
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(24 * time.Hour * time.Duration(TokenExpiry)).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tokenSecret))
}

func ParseToken(token string) (*Claims, error) {
	jwt, err := jwt.ParseWithClaims(token, &Claims{}, keyFunc)
	if err != nil {
		return nil, err
	}

	claims, ok := jwt.Claims.(*Claims)
	if !ok {
		return nil, errors.New("claims invalid")
	}

	return claims, nil
}

func IsSponsor(login string) (bool, error) {
	mux.Lock()
	defer mux.Unlock()

	if _, ok := authorized[login]; !ok {
		if time.Since(updated) < updateInterval {
			return false, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		all, err := sponsor.Get(ctx)
		if err != nil {
			return false, err
		}

		updated = time.Now()

		for _, s := range all {
			authorized[s.Login] = true
		}
	}

	_, ok := authorized[login]
	return ok, nil
}
