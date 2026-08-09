package vehicle

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

// TeslaFleetConfig contains Tesla Fleet API credentials and tokens
type TeslaFleetConfig struct {
	Credentials ClientCredentials
	Tokens      Tokens
}

// TeslaFleetClient provides authenticated access to the Tesla Fleet API
type TeslaFleetClient struct {
	Client     *teslaclient.Client
	HTTPClient *http.Client
}

// Validate checks that the required Tesla Fleet API credentials are configured
func (c TeslaFleetConfig) Validate() error {
	if c.Credentials.ID == "" {
		return errors.New("missing client id, see https://docs.evcc.io/en/docs/devices/vehicles#tesla")
	}
	if c.Tokens.Access == "" || c.Tokens.Refresh == "" {
		return api.ErrMissingToken
	}

	return nil
}

// Client creates a Tesla Fleet API client for the configured account
func (c TeslaFleetConfig) Client(log *util.Logger) (*TeslaFleetClient, error) {
	token, err := c.Tokens.Token()
	if err != nil {
		return nil, err
	}

	identity, err := tesla.NewIdentity(log, tesla.OAuth2Config(c.Credentials.ID, c.Credentials.Secret), token)
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

	return &TeslaFleetClient{Client: tc, HTTPClient: hc}, nil
}
