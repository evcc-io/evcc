package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
)

func TokenRefreshServiceTokenSource(log *util.Logger, q url.Values, user, password string) (vag.TokenSource, error) {
	return vwidentity.TokenSource(log, q, user, password)
	// q, err := vwidentity.Login(log, q, user, password)
	// if err != nil {
	// 	return nil, err
	// }

	// trs := tokenrefreshservice.New(log, data)
	// token, err := trs.Exchange(q)
	// if err != nil {
	// 	return nil, err
	// }

	// return trs.TokenSource(token), nil
}
