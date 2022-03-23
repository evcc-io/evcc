package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag/mbb"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

func MbbTokenSource(log *util.Logger, clientID string, q url.Values, user, password string) (oauth2.TokenSource, error) {
	// verify := vag.ChallengeAndVerifier(q)

	vwi := vwidentity.New(log)
	uri := vwidentity.LoginURL(vwidentity.Endpoint.AuthURL, q)
	q, err := vwi.Login(uri, user, password)
	if err != nil {
		return nil, err
	}

	mbb := mbb.New(log, clientID)
	token, err := mbb.Exchange(q)
	if err != nil {
		return nil, err
	}

	// verify(q)

	// trs := tokenrefreshservice.New(log)
	// token, err := trs.Exchange(q)
	// if err != nil {
	// 	return nil, err
	// }

	return mbb.TokenSource(token), nil
}
