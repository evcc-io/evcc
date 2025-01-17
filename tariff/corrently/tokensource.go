package corrently

import (
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	log *util.Logger
}

func TokenSource(log *util.Logger, token *oauth2.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(token, &tokenSource{log})
}

func (ts *tokenSource) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	//	"Content-Type: application/json" \
	// --request POST \
	// https://console.corrently.io/v2.0/auth/requestToken

	var res struct {
		Token   string `json:"token"`
		Expires int64  `json:"expires"`
	}

	req, _ := request.New(http.MethodPost, "https://console.corrently.io/v2.0/auth/requestToken", nil, request.JSONEncoding)
	err := request.NewHelper(ts.log).DoJSON(req, &res)

	token := &oauth2.Token{
		AccessToken: res.Token,
		Expiry:      time.UnixMilli(res.Expires),
	}

	return token, err
}
