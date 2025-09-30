package cardata

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

var ErrLoginRequired = errors.New("login required")

var Config = oauth2.Config{
	Scopes: []string{"authenticate_user", "openid", "cardata:streaming:read", "cardata:api:read"},
	Endpoint: oauth2.Endpoint{
		DeviceAuthURL: "https://customer.bmwgroup.com/gcdm/oauth/device/code",
		TokenURL:      "https://customer.bmwgroup.com/gcdm/oauth/token",
	},
}

var (
	idMu       sync.Mutex
	identities = make(map[string]oauth2.TokenSource)
)

type Identity struct {
	oauth2.Config
	mu  sync.Mutex
	ts  oauth2.TokenSource
	log *util.Logger
}

// NewIdentity creates BMW/Mini Cardata identity
func NewIdentity(ctx context.Context, log *util.Logger, clientID string) (oauth2.TokenSource, error) {
	idMu.Lock()
	defer idMu.Unlock()

	if id, ok := identities[clientID]; ok {
		return id, nil
	}

	v := &Identity{
		log:    log,
		Config: Config,
	}

	v.ClientID = clientID

	var cardataToken Token
	if err := settings.Json(v.settingsKey(), &cardataToken); err == nil {
		v.log.DEBUG.Println("database token found")

		token := cardataToken.TokenEx()

		ctx := context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))
		v.ts = &PersistingTokenSource{
			TokenSource: Config.TokenSource(ctx, token),
			Persist:     v.storeToken,
		}
	} else {
		v.log.DEBUG.Println("no database token found, login required")

		return nil, ErrLoginRequired
	}

	// store
	identities[clientID] = v

	return v, nil
}

func (v *Identity) Token() (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.ts == nil {
		return nil, ErrLoginRequired
	}

	return v.ts.Token()
}

func (v *Identity) storeToken(token *oauth2.Token) error {
	cardataToken := &Token{
		Token:   token,
		IdToken: tokenExtra(token, "id_token"),
		Gcid:    tokenExtra(token, "gcid"),
	}

	return settings.SetJson(v.settingsKey(), cardataToken)
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("cardata-%s", v.Config.ClientID)
}
