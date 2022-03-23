package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/audi"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/idkproxy"
	"github.com/evcc-io/evcc/vehicle/vag/mbb"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
)

func MbbTokenSource(log *util.Logger, clientID string, q url.Values, user, password string) (vag.TokenSource, error) {
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

	return mbb.TokenSource(token), nil
}

func MbbIDKTokenSource(log *util.Logger, clientID string, q url.Values, user, password string) (vag.TokenSource, error) {
	verify := vag.ChallengeAndVerifier(q)

	vwi := vwidentity.New(log)
	uri := vwidentity.LoginURL(vwidentity.Endpoint.AuthURL, q)
	q, err := vwi.Login(uri, user, password)
	if err != nil {
		return nil, err
	}

	verify(q)

	// trs := tokenrefreshservice.New(log)
	// token, err := trs.Exchange(q)
	// if err != nil {
	// 	return nil, err
	// }

	idk := idkproxy.New(log, audi.IDKParams)
	token, err := idk.Exchange(q)
	if err != nil {
		return nil, err
	}

	// token, err = idk.Refresh(token)
	// if err != nil {
	// 	return nil, err
	// }

	// token.Expiry = time.Now()
	// token, err = idk.TokenSource(token).Token()
	// if err != nil {
	// 	return nil, err
	// }

	// azs := aazsproxy.New(log)
	// token, err = azs.Exchange(url.Values{"id_token": {token.AccessToken}})
	// if err != nil {
	// 	return nil, err
	// }

	mbb := mbb.New(log, clientID)
	token, err = mbb.Exchange(url.Values{"id_token": {token.IDToken}})
	if err != nil {
		return nil, err
	}

	return mbb.TokenSource(token), nil
}
