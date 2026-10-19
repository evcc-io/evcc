package tesla

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("tesla", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID, ClientSecret string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		log := util.NewLogger("tesla").Redact(cc.ClientID, cc.ClientSecret)
		ctx := util.WithLogger(context.Background(), log)

		return NewOAuth(ctx, cc.ClientID, cc.ClientSecret)
	})
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

var registerMu sync.Mutex

// NewOAuth creates the Fleet API token source for the public url of this instance. Login registers its host with Tesla first.
func NewOAuth(ctx context.Context, clientID, clientSecret string) (oauth2.TokenSource, error) {
	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("tesla"))

	origin := network.PublicURL()
	if origin == "" {
		return nil, errors.New("missing public url, enable remote access or set an https external url")
	}
	domain := strings.TrimPrefix(origin, "https://")

	return auth.NewOAuth(ctx, "Tesla", "", OAuthConfig(clientID, clientSecret, origin),
		// region is resolved from the account after login
		auth.WithExchangeOptions(oauth2.SetAuthURLParam("audience", FleetBaseUrl)),
		auth.WithLoginHook(func(ctx context.Context) error {
			// concurrent logins must not replace the key tesla is fetching
			registerMu.Lock()
			defer registerMu.Unlock()

			if _, err := ensurePrivateKey(); err != nil {
				return err
			}

			err := registerPartner(ctx, log, clientID, clientSecret, domain)
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
