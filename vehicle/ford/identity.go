package ford

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	AuthURI  = "https://sso.ci.ford.ca" // fcis.ice.ibmcloud.com
	ClientID = "9fb503e0-715b-47e8-adfd-ad4b7770f73b"
)

type Identity struct {
	*request.Helper
	user, password string
	oauth2.TokenSource
}

// NewIdentity creates Fiat identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// Login authenticates with username/password to get new aws credentials
func (v *Identity) Login() error {
	token, err := v.login()
	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource((*oauth2.Token)(&token), v)
	}
	return err
}

// login authenticates with username/password to get new token
func (v *Identity) login() (oauth.Token, error) {
	data := url.Values{
		"client_id":  []string{ClientID},
		"grant_type": []string{"password"},
		"username":   []string{v.user},
		"password":   []string{v.password},
	}

	uri := AuthURI + "/v1.0/endpoint/default/token"
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	var res oauth.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// Refresh implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":     []string{ClientID},
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{token.RefreshToken},
	}

	uri := AuthURI + "/v1.0/endpoint/default/token"
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	var res oauth.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err != nil {
		res, err = v.login()
	}

	return (*oauth2.Token)(&res), err
}
