package tronity

import (
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	log *util.Logger
	oc  *oauth2.Config
}

func TokenSource(log *util.Logger, oc *oauth2.Config) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &tokenSource{log, oc})
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	data := struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
	}{
		ClientID:     ts.oc.ClientID,
		ClientSecret: ts.oc.ClientSecret,
		GrantType:    "app",
	}

	req, _ := request.New(http.MethodPost, ts.oc.Endpoint.TokenURL, request.MarshalJSON(data), request.JSONEncoding)

	var token oauth2.Token
	err := request.NewHelper(ts.log).DoJSON(req, &token)

	return util.TokenWithExpiry(&token), err
}
