package tesla

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// FleetConfig contains Tesla Fleet API credentials and tokens.
// Without tokens, they are obtained interactively via the evcc UI using the client secret.
type FleetConfig struct {
	Credentials oauth.ClientCredentials
	Tokens      oauth.Tokens
	Origin      string // public https address, defaults to network.PublicOrigin
}

// Interactive returns true when tokens are obtained via UI login instead of static config
func (c FleetConfig) Interactive() bool {
	return c.Tokens.Access == "" && c.Tokens.Refresh == "" && c.Credentials.Secret != ""
}

// Validate checks that the required Tesla Fleet API credentials are configured
func (c FleetConfig) Validate() error {
	if c.Credentials.ID == "" {
		return errors.New("missing client id, see https://docs.evcc.io/en/docs/devices/vehicles#tesla")
	}

	if c.Interactive() {
		if c.Origin == "" {
			return nil
		}
		_, err := originHost(c.Origin)
		return err
	}

	if c.Tokens.Access == "" || c.Tokens.Refresh == "" {
		return api.ErrMissingToken
	}

	return nil
}

func (c FleetConfig) tokenSource(log *util.Logger) (oauth2.TokenSource, error) {
	if c.Interactive() {
		ctx := util.WithLogger(context.Background(), log)
		return NewOAuth(ctx, c.Credentials.ID, c.Credentials.Secret, c.Origin)
	}

	token, err := c.Tokens.Token()
	if err != nil {
		return nil, err
	}

	return NewIdentity(log, OAuth2Config(c.Credentials.ID, c.Credentials.Secret), token)
}

// Client creates a Tesla Fleet API client for the configured account
func (c FleetConfig) Client(log *util.Logger) (*API, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	identity, err := c.tokenSource(log)
	if err != nil {
		return nil, fmt.Errorf("create Fleet identity: %w", err)
	}

	hc := request.NewClient(log)
	hc.Transport = &oauth2.Transport{Source: identity, Base: hc.Transport}

	region, err := NewAPI(hc, FleetBaseUrl).Region()
	if err != nil {
		return nil, fmt.Errorf("get Fleet API region: %w", err)
	}

	return NewAPI(hc, region.FleetApiBaseUrl), nil
}
