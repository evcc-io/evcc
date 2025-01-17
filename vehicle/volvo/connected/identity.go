package connected

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

type oauth2Config struct {
	*oauth2.Config
	h *request.Helper
}

func Oauth2Config(log *util.Logger, id, secret, redirect string) *oauth2Config {
	return &oauth2Config{
		h: request.NewHelper(log),
		Config: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			RedirectURL:  redirect,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
				TokenURL: "https://volvoid.eu.volvocars.com/as/token.oauth2",
			},
			Scopes: []string{
				oidc.ScopeOpenID,
				// "vehicle:attributes",
				"energy:recharge_status", "energy:battery_charge_level", "energy:electric_range", "energy:estimated_charging_time", "energy:charging_connection_status", "energy:charging_system_status",
				// "conve:fuel_status", "conve:odometer_status", "conve:environment",
			},
		},
	}
}

// func (oc *oauth2Config) TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
// 	return oauth.RefreshTokenSource(token, oc)
// }

type Identity struct {
	*request.Helper
	oc            *oauth2Config
	ts            oauth2.TokenSource
	mu            sync.Mutex
	cv            string
	authenticated chan<- bool
}

func NewIdentity(log *util.Logger, config *oauth2Config) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		oc:     config,
	}
}

var _ api.AuthProvider = (*Identity)(nil)

// func (v *Identity) SetCallbackParams(baseURL, redirectURL string, authenticated chan<- bool) {
// 	v.authenticated = authenticated
// 	fmt.Println(baseURL, redirectURL)
// }

func (v *Identity) LoginHandler(authenticated chan<- bool) http.HandlerFunc {
	v.authenticated = authenticated

	return func(w http.ResponseWriter, req *http.Request) {
		v.cv = oauth2.GenerateVerifier()
		state := lo.RandomString(16, lo.AlphanumericCharset)
		url := v.oc.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(v.cv))
		http.Redirect(w, req, url, http.StatusTemporaryRedirect)
	}
}

func (v *Identity) RedirectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		token, err := v.oc.Exchange(r.Context(), code, oauth2.VerifierOption(v.cv))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (v *Identity) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		defer v.mu.Unlock()
		v.ts = nil
	}
}

var _ oauth2.TokenSource = (*Identity)(nil)

func (v *Identity) Token() (*oauth2.Token, error) {
	return nil, api.ErrMissingToken
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		// "access_token_manager_id": {managerId},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	req, err := request.New(http.MethodPost, v.oc.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		// "Authorization": basicAuth,
	})
	if err != nil {
		return nil, err
	}

	var res oauth2.Token
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return util.TokenWithExpiry(&res), err
}
