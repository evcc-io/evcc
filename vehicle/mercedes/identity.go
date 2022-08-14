package mercedes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type IdentityOptions func(c *Identity) error

// WithToken provides an oauth2.Token to the client for auth.
func WithToken(t *oauth2.Token) IdentityOptions {
	return func(v *Identity) error {
		v.ReuseTokenSource.Apply(t)
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

// TODO SessionSecret from config/persistence
func NewIdentity(log *util.Logger, id, secret string, options ...IdentityOptions) (*Identity, error) {
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

	ts := &ReuseTokenSource{
		oc: oc,
		cb: v.invalidToken,
	}
	ts.Apply(nil)

	v.ReuseTokenSource = ts

	for _, o := range options {
		if err == nil {
			err = o(v)
		}
	}

	return v, err
}

// invalidToken is the callback for the token source when token expires
func (v *Identity) invalidToken() {
	if v.authC != nil {
		v.authC <- false
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
		v.ReuseTokenSource.Apply(nil)
		v.authC <- false

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nil)
	}
}

func (v *Identity) callbackHandler(w http.ResponseWriter, r *http.Request) {
	v.log.TRACE.Println("callback request retrieved")

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
		v.ReuseTokenSource.Apply(token)
		v.authC <- true

		provider.ResetCached()
	}

	http.Redirect(w, r, v.baseURL, http.StatusFound)
}
