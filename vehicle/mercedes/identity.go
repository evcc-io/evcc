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
	"github.com/gorilla/mux"
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
	// tokenSource oauth2.TokenSource
	router *mux.Router

	loginUpdateC chan struct{}
	apiPath      string
}

// TODO: SessionSecret from config/persistence
func NewIdentity(log *util.Logger, id, secret string, loginUpdateC chan struct{}, options ...IdentityOptions) (*Identity, error) {
	var err error
	provider, err := oidc.NewProvider(context.Background(), "https://id.mercedes-benz.com")
	if err != nil {
		log.FATAL.Printf("failed to inizialize OIDC provider: %s", err)
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
			// TODO: configure properly redirectURL
			RedirectURL: "http://localhost:7070/vehicle/mercedes/callback",
		},
		apiPath: "/identityproviders/mercedes",
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

var _ api.WebController = (*Identity)(nil)

func (v *Identity) WebControl(router *mux.Router) {
	v.router = router

	v.router.HandleFunc("/vehicle/mercedes/callback", v.redirectHandler(context.Background()))

	v.router.Methods(http.MethodPost).Path(v.LoginPath()).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})

	v.router.Methods(http.MethodPost).Path(v.LogoutPath()).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v.token = nil

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nil)
	})
}

// LoggedIn implements the api.ProviderLogin interface
func (v *Identity) LoggedIn() bool {
	return v.token.Valid()
}

// LoginPath implements the api.ProviderLogin interface
func (v *Identity) LoginPath() string {
	return fmt.Sprintf("%s/login", v.apiPath)
}

// LogoutPath implements the api.ProviderLogin interface
func (v *Identity) LogoutPath() string {
	return fmt.Sprintf("%s/logout", v.apiPath)
}

func (v *Identity) redirectHandler(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		token, err := v.AuthConfig.Exchange(ctx, codes[0])
		if err != nil {
			fmt.Fprintln(w, "token error:", err)
			return
		}

		if token.Valid() {
			v.token = token
			v.log.TRACE.Println("sending login update...")
			v.loginUpdateC <- struct{}{}
		}

		// TODO: make uri configurable like v.LocalURI = "http://localhost:7070"
		http.Redirect(w, r, "http://localhost:7070", http.StatusFound)
	}
}
