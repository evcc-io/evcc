package tesla

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("tesla", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID, ClientSecret, Origin string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		log := util.NewLogger("tesla").Redact(cc.ClientID, cc.ClientSecret)
		ctx := util.WithLogger(context.Background(), log)

		return NewOAuth(ctx, cc.ClientID, cc.ClientSecret, cc.Origin)
	})
}

// originHost returns the host of the public https origin, which is the domain registered with Tesla
func originHost(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("invalid origin, expected https://host: %s", origin)
	}
	return u.Host, nil
}

// OAuthConfig returns the Fleet API OAuth2 config for a third-party app
func OAuthConfig(clientID, clientSecret, origin string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  origin + network.CallbackPath,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		// scopes cannot be widened later without revoking the app, energy scopes cover a later Powerwall
		Scopes: []string{
			"openid", "offline_access", "user_data",
			"vehicle_device_data", "vehicle_location", "vehicle_cmds", "vehicle_charging_cmds",
			"energy_device_data", "energy_cmds",
		},
	}
}

// NewOAuth creates the Fleet API token source. Login registers the origin host with Tesla first.
// An empty origin defaults to the remote access url.
func NewOAuth(ctx context.Context, clientID, clientSecret, origin string) (oauth2.TokenSource, error) {
	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("tesla"))

	if origin == "" {
		if origin = remoteOrigin(); origin == "" {
			return nil, errors.New("missing origin, enable remote access or configure a public https address")
		}
	}

	return auth.NewOAuth(ctx, "Tesla", "", OAuthConfig(clientID, clientSecret, origin),
		// region is resolved from the account after login
		auth.WithExchangeOptions(oauth2.SetAuthURLParam("audience", strings.TrimSuffix(teslaclient.FleetAudienceNA, "/"))),
		auth.WithLoginHook(func(ctx context.Context) error {
			domain, err := originHost(origin)
			if err != nil {
				return err
			}
			if _, err := ensurePrivateKey(); err != nil {
				return err
			}

			err = registerPartner(ctx, log, clientID, clientSecret, domain)
			if errors.Is(err, errKeyTaken) {
				log.WARN.Printf("signing key is registered for another app or domain, replacing it, vehicles must pair again")
				if _, err := generatePrivateKey(); err != nil {
					return err
				}
				err = registerPartner(ctx, log, clientID, clientSecret, domain)
			}

			return err
		}),
	)
}
