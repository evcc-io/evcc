package etron

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const TokenURI = "https://aazsproxy-service.apps.emea.vwapps.io/token"

// Identity is the etron AZS token client
type Identity struct {
	*request.Helper
	token oauth2.Token
}

// NewIdentity creates a new Identity client
func NewIdentity(log *util.Logger, ts oauth2.TokenSource) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
	}

	token, err := ts.Token()
	if err != nil {
		panic(err)
	}

	data := struct {
		Token     string `json:"token"`
		GrantType string `json:"grant_type"`
		Stage     string `json:"stage"`
		Config    string `json:"config"`
	}{
		Token:     token.AccessToken,
		GrantType: "id_token",
		Stage:     "live",
		Config:    "myaudi",
	}

	var req *http.Request
	req, err = request.New(http.MethodPost, TokenURI, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &v.token)
	}

	if err != nil {
		panic(err)
	}

	return v
}

func (v *Identity) Token() (*oauth2.Token, error) {
	var err error
	if !v.token.Valid() {
		err = errors.New("invalid")
	}

	return &v.token, err
}
