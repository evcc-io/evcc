package graphql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// ErrAuthFailed indicates the Kraken API rejected the supplied credentials.
var ErrAuthFailed = errors.New("authentication failed")

type tokenSource struct {
	log             *util.Logger
	email, password string
}

var _ oauth2.TokenSource = (*tokenSource)(nil)

// Token implements oauth2.TokenSource to obtain a JWT from the EDF UK Kraken API.
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cli := request.NewClient(ts.log)
	tempClient := graphql.NewClient(BaseURI, cli)

	var q krakenTokenAuthentication
	if err := tempClient.Mutate(ctx, &q, map[string]any{
		"email":    ts.email,
		"password": ts.password,
	}); err != nil {
		if _, ok := errors.AsType[graphql.Errors](err); ok {
			return nil, fmt.Errorf("%w: %v", ErrAuthFailed, err)
		}
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(q.ObtainKrakenToken.Token, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	expiry := time.Now().Add(time.Hour)
	if claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}

	return &oauth2.Token{
		AccessToken: q.ObtainKrakenToken.Token,
		Expiry:      expiry,
	}, nil
}
