package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
)

func TokenRefreshServiceTokenSource(log *util.Logger, toxValues, q url.Values, user, password string) (vag.TokenSource, error) {
	q, err := vwidentity.Login(log, q, user, password)
	if err != nil {
		return nil, err
	}

	trs := tokenrefreshservice.New(log, toxValues)
	token, err := trs.Exchange(q)
	if err != nil {
		return nil, err
	}

	return trs.TokenSource(token), nil
}
