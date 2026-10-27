package tesla

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// FleetConfig contains Tesla Fleet API credentials and tokens
type FleetConfig struct {
	Credentials oauth.ClientCredentials
	Tokens      oauth.Tokens
}

// Validate checks that the required Tesla Fleet API credentials are configured
func (c FleetConfig) Validate() error {
	if c.Credentials.ID == "" {
		return errors.New("missing client id, see https://docs.evcc.io/en/docs/devices/vehicles#tesla")
	}
	if c.Tokens.Access == "" || c.Tokens.Refresh == "" {
		return api.ErrMissingToken
	}

	return nil
}

// Client creates a Tesla Fleet API client for the configured account
func (c FleetConfig) Client(log *util.Logger) (*API, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	token, err := c.Tokens.Token()
	if err != nil {
		return nil, err
	}

	identity, err := NewIdentity(log, OAuth2Config(c.Credentials.ID, c.Credentials.Secret), token)
	if err != nil {
		return nil, fmt.Errorf("create Fleet identity: %w", err)
	}

	hc := request.NewClient(log)
	hc.Transport = &oauth2.Transport{Source: identity, Base: hc.Transport}

	region, err := NewAPI(log, hc, FleetBaseUrl).Region()
	if err != nil {
		return nil, fmt.Errorf("get Fleet API region: %w", err)
	}

	return NewAPI(log, hc, region.FleetApiBaseUrl), nil
}
