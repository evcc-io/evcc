package tesla

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

// FleetConfig contains Tesla Fleet API credentials and tokens.
type FleetConfig struct {
	Credentials struct{ ID, Secret string }
	Tokens      struct{ Access, Refresh string }
}

// FleetClient provides authenticated access to the Tesla Fleet API.
type FleetClient struct {
	Client     *teslaclient.Client
	HTTPClient *http.Client
}

// Validate checks that the required Tesla Fleet API credentials are configured.
func (c FleetConfig) Validate() error {
	if c.Credentials.ID == "" {
		return errors.New("missing client id")
	}
	if c.Tokens.Access == "" || c.Tokens.Refresh == "" {
		return api.ErrMissingToken
	}

	return nil
}

// Client creates a Tesla Fleet API client for the configured account.
func (c FleetConfig) Client(log *util.Logger) (*FleetClient, error) {
	identity, err := NewIdentity(log, OAuth2Config(c.Credentials.ID, c.Credentials.Secret), &oauth2.Token{
		AccessToken:  c.Tokens.Access,
		RefreshToken: c.Tokens.Refresh,
		Expiry:       time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("create Fleet identity: %w", err)
	}

	hc := request.NewClient(log)
	hc.Transport = &oauth2.Transport{Source: identity, Base: hc.Transport}

	tc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(hc))
	if err != nil {
		return nil, fmt.Errorf("create Fleet client: %w", err)
	}

	region, err := tc.UserRegion()
	if err != nil {
		return nil, fmt.Errorf("get Fleet API region: %w", err)
	}
	tc.SetBaseUrl(region.FleetApiBaseUrl)

	return &FleetClient{Client: tc, HTTPClient: hc}, nil
}
