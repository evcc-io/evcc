package mercedes

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type IdentityOptions func(c *Identity) error

// WithToken provides an oauth2.Token to the client for auth.
func WithToken(t *oauth2.Token) IdentityOptions {
	return func(c *Identity) error {
		c.token = t
		return nil
	}
}

type Identity struct {
	log *util.Logger

	sessionSecret []byte

	AuthConfig *oauth2.Config
	token      *oauth2.Token

	loginUpdateC chan struct{}
	basePath     string
}

// TODO: SessionSecret from config/persistence
func NewIdentity(log *util.Logger, id, secret string, loginUpdateC chan struct{}, options ...IdentityOptions) (*Identity, error) {
	var err error
	provider, err := oidc.NewProvider(context.Background(), "https://id.mercedes-benz.com")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %s", err)
	}

	v := &Identity{
		log:           log,
		loginUpdateC:  loginUpdateC,
		sessionSecret: genSessionSecret(),
		AuthConfig: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOfflineAccess, "mb:vehicle:mbdata:evstatus"},
		},
	}

	for _, o := range options {
		if err == nil {
			err = o(v)
		}
	}

	return v, err
}

func genSessionSecret() []byte {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return b[:]
}

func (v *Identity) Token() *oauth2.Token {
	return v.token
}

var _ api.ProviderLogin = (*Identity)(nil)

func (v *Identity) SetBasePath(basepath string) {
	v.basePath = basepath
}

func (v *Identity) Callback() api.Callback {
	return api.Callback{
		Path:    fmt.Sprintf("%s/callback", v.basePath),
		Handler: v.redirectHandler,
	}
}

func (v *Identity) SetOAuthCallbackURI(uri string) {
	v.AuthConfig.RedirectURL = uri
}

func (v *Identity) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := NewState(v.sessionSecret)

		b, _ := json.Marshal(struct {
			LoginUri string `json:"loginUri"`
		}{
			LoginUri: v.AuthConfig.AuthCodeURL(state.Encrypt(), oauth2.AccessTypeOffline,
				oauth2.SetAuthURLParam("prompt", "login consent"),
			),
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func (v *Identity) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v.token = nil

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nil)
	}
}

// LoggedIn implements the api.ProviderLogin interface
func (v *Identity) LoggedIn() bool {
	return v.token.Valid()
}

// LoginPath implements the api.ProviderLogin interface
func (v *Identity) LoginPath() string {
	return fmt.Sprintf("%s/login", v.basePath)
}

// LogoutPath implements the api.ProviderLogin interface
func (v *Identity) LogoutPath() string {
	return fmt.Sprintf("%s/logout", v.basePath)
}

func (v *Identity) redirectHandler(uri string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v.log.TRACE.Println("callback request retrieved")

		data, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			fmt.Fprintln(w, "invalid response:", data)
			return
		}

		if error, ok := data["error"]; ok {
			fmt.Fprintf(w, "error: %s: %s\n", error, data["error_description"])
			return
		}

		states, ok := data["state"]
		if !ok || len(states) != 1 {
			fmt.Fprintln(w, "invalid state response:", data)
			return
		} else if err := Validate(states[0], v.sessionSecret); err != nil {
			fmt.Fprintf(w, "failed state validation: %s", err)
			return
		}

		codes, ok := data["code"]
		if !ok || len(codes) != 1 {
			fmt.Fprintln(w, "invalid response:", data)
			return
		}

		token, err := v.AuthConfig.Exchange(context.Background(), codes[0])
		if err != nil {
			fmt.Fprintln(w, "token error:", err)
			return
		}

		if token.Valid() {
			v.token = token
			v.log.TRACE.Println("sending login update...")
			v.loginUpdateC <- struct{}{}
		}

		http.Redirect(w, r, uri, http.StatusFound)
	}
}
