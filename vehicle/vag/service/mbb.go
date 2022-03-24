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
	q, err := vwidentity.Login(log, q, user, password)
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
	q, err := vwidentity.Login(log, q, user, password)
	if err != nil {
		return nil, err
	}

	idk := idkproxy.New(log, audi.IDKParams)
	token, err := idk.Exchange(q)
	if err != nil {
		return nil, err
	}

	its := idk.TokenSource(token)
	mbb := mbb.New(log, clientID)

	mts := vag.MetaTokenSource(func() (*vag.Token, error) {
		// get IDK token from refreshing IDK token source
		itoken, err := its.TokenEx()
		if err != nil {
			return nil, err
		}

		// exchange IDK id_token for MBB token
		mtoken, err := mbb.Exchange(url.Values{"id_token": {itoken.IDToken}})
		if err != nil {
			return nil, err
		}

		return mtoken, err

		// produce tokens from refresh MBB token source
	}, mbb.TokenSource)

	return mts, nil
}
