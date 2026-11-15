package tesla

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/golang-jwt/jwt/v5"
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

	// the identity may hold a newer token than the configured one
	if current, err := identity.Token(); err == nil {
		token = current
	}

	region, err := NewAPI(log, hc, fleetBaseUrl(token.AccessToken)).Region()
	if err != nil {
		return nil, fmt.Errorf("get Fleet API region: %w", err)
	}

	return NewAPI(log, hc, region.FleetApiBaseUrl), nil
}

// fleetBaseUrl returns the Fleet API base url the access token is issued for.
// An app registered in a single region is rejected by the other regions, including for the region lookup.
func fleetBaseUrl(accessToken string) string {
	var claims struct {
		OuCode string `json:"ou_code"`
		jwt.RegisteredClaims
	}
	if _, _, err := jwt.NewParser().ParseUnverified(accessToken, &claims); err != nil {
		return FleetBaseUrl
	}

	res := FleetBaseUrl
	region := "." + strings.ToLower(claims.OuCode) + "."

	for _, aud := range claims.Audience {
		u, err := url.Parse(aud)
		if err != nil || u.Scheme != "https" || !strings.HasPrefix(u.Host, "fleet-api.") {
			continue
		}
		if !strings.HasSuffix(u.Host, ".tesla.com") && !strings.HasSuffix(u.Host, ".tesla.cn") {
			continue
		}

		res = "https://" + u.Host
		if strings.Contains(u.Host, region) {
			break
		}
	}

	return res
}
