package mercedes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type IdentityOption func(c *Identity) error

// WithStore provides an oauth2.Token to the client for auth.
func WithStore(store store.Store) IdentityOption {
	return func(v *Identity) error {
		if store != nil && !reflect.ValueOf(store).IsNil() {
			v.ReuseTokenSource.WithStore(store)
		}
		return nil
	}
}

type Identity struct {
	log *util.Logger
	*ReuseTokenSource
	oc      *oauth2.Config
	baseURL string
	authC   chan<- bool
}

func NewIdentity(log *util.Logger, id, secret string, options ...IdentityOption) (*Identity, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://id.mercedes-benz.com")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %s", err)
	}

	oc := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOfflineAccess, "mb:vehicle:mbdata:evstatus"},
	}

	v := &Identity{
		log: log,
		oc:  oc,
	}

	v.ReuseTokenSource = &ReuseTokenSource{
		oc: oc,
		cb: v.loggedIn,
	}

	for _, o := range options {
		if err == nil {
			err = o(v)
		}
	}

	var token *oauth2.Token
	_ = v.store.Load(&token)

	v.Update(token)

	return v, err
}

// loggedIn is the callback for the token source when token expires
func (v *Identity) loggedIn(val bool) {
	if v.authC != nil {
		v.authC <- val
	}
}

var _ api.AuthProvider = (*Identity)(nil)

func (v *Identity) SetCallbackParams(baseURL, redirectURL string, authC chan<- bool) {
	v.baseURL = baseURL
	v.oc.RedirectURL = redirectURL
	v.authC = authC
}

func (v *Identity) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := auth.Register(v.callbackHandler)

		b, _ := json.Marshal(struct {
			LoginUri string `json:"loginUri"`
		}{
			LoginUri: v.oc.AuthCodeURL(state, oauth2.AccessTypeOffline,
				oauth2.SetAuthURLParam("prompt", "login consent"),
			),
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func (v *Identity) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v.log.TRACE.Println("sending logout update...")
		v.ReuseTokenSource.Update(nil)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nil)
	}
}

func (v *Identity) callbackHandler(w http.ResponseWriter, r *http.Request) {
	v.log.DEBUG.Println("callback request received")

	data, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		fmt.Fprintln(w, "invalid response:", data)
		return
	}

	codes, ok := data["code"]
	if !ok || len(codes) != 1 {
		fmt.Fprintln(w, "invalid response:", data)
		return
	}

	token, err := v.oc.Exchange(context.Background(), codes[0])
	if err != nil {
		fmt.Fprintln(w, "token error:", err)
		return
	}

	if token.Valid() {
		v.log.TRACE.Println("sending login update...")
		v.ReuseTokenSource.Update(token)

		provider.ResetCached()
	}

	http.Redirect(w, r, v.baseURL, http.StatusFound)
}
