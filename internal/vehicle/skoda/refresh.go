package skoda

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/oauth"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

type tokenRefresher struct {
	*request.Helper
}

func Refresher(log *util.Logger) oauth.TokenRefresher {
	return &tokenRefresher{
		Helper: request.NewHelper(log),
	}
}

// RefreshToken implements oauth.TokenRefresher
func (tr *tokenRefresher) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	uri := "https://tokenrefreshservice.apps.emea.vwapps.io/refreshTokens"

	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"brand":         {"skoda"},
	})

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	var res oauth.Token
	if err == nil {
		err = tr.DoJSON(req, &res)
	}

	return (*oauth2.Token)(&res), err
}
