package skoda

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

func TokenRefreshServiceTokenSource(log *util.Logger, q url.Values, user, password string) (oauth2.TokenSource, error) {
	vwi := vwidentity.New(log)
	uri := vwidentity.LoginURL(vwidentity.Endpoint.AuthURL, q)
	q, err := vwi.Login(uri, user, password)
	if err != nil {
		return nil, err
	}

	data := url.Values{
		"brand": {"skoda"},
	}

	trs := tokenrefreshservice.New(log, data)
	token, err := trs.Exchange(q)
	if err != nil {
		return nil, err
	}

	// TODO add brand param
	return trs.TokenSource(token), nil
}
