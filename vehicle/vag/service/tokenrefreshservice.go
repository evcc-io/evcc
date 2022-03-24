package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

func TokenRefreshServiceTokenSource(log *util.Logger, q, data url.Values, user, password string) (oauth2.TokenSource, error) {
	q, err := vwidentity.Login(log, q, user, password)
	if err != nil {
		return nil, err
	}

	trs := tokenrefreshservice.New(log, data)
	token, err := trs.Exchange(q)
	if err != nil {
		return nil, err
	}

	// TODO add brand param
	return trs.TokenSource(token), nil
}
