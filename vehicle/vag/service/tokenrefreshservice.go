package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util/log"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

func TokenRefreshServiceTokenSource(log log.Logger, data, q url.Values, user, password string) (oauth2.TokenSource, error) {
	q, err := vwidentity.Login(log, q, user, password)
	if err != nil {
		return nil, err
	}

	trs := tokenrefreshservice.New(log, data)
	token, err := trs.Exchange(q)
	if err != nil {
		return nil, err
	}

	return trs.TokenSource(token), nil
}
