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
// Callers should treat this as permanent and stop retrying to avoid account lockouts.
var ErrAuthFailed = errors.New("authentication failed")

type tokenSource struct {
	log             *util.Logger
	email, password string
}

var _ oauth2.TokenSource = (*tokenSource)(nil)

// RefreshToken implements oauth.TokenRefresher to obtain a new JWT token.
// It parses the JWT to extract the actual expiry time from the token claims.
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create a temporary client without authentication for the token request
	cli := request.NewClient(ts.log)
	tempClient := graphql.NewClient(BaseURI, cli)

	var q krakenTokenAuthentication
	if err := tempClient.Mutate(ctx, &q, map[string]any{
		"email":    ts.email,
		"password": ts.password,
	}); err != nil {
		// Any GraphQL error response from obtainKrakenToken is an application-level
		// rejection (bad credentials, account locked, etc.) — repeating the request
		// will not change the outcome and continued retries can lock the account.
		// Network/transport failures don't surface as graphql.Errors and stay
		// transient via the wrapped path below.
		if _, ok := errors.AsType[graphql.Errors](err); ok {
			return nil, fmt.Errorf("%w: %v", ErrAuthFailed, err)
		}
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Parse JWT to extract expiry time using RegisteredClaims
	// We use ParseUnverified since we don't have the signing key and trust the token from the API
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(q.ObtainKrakenToken.Token, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	// Extract expiry from JWT claims
	expiry := time.Now().Add(time.Hour)
	if claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}

	return &oauth2.Token{
		AccessToken: q.ObtainKrakenToken.Token,
		Expiry:      expiry,
	}, nil
}
