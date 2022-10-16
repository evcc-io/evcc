package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/mbb"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
)

// MbbTokenSource creates a refreshing token source for use with the MBB api.
// Once the MBB token expires, it is recreated from the token exchanger (either TokenRefreshService or IDK)
func MbbTokenSource(log *util.Logger, tox vag.TokenExchanger, mbb *mbb.Service, q url.Values, user, password string) (vag.TokenSource, error) {
	trs := tox.TokenSource(nil)
	if trs == nil {
		q, err := vwidentity.Login(log, q, user, password)
		if err != nil {
			return nil, err
		}

		token, err := tox.Exchange(q)
		if err != nil {
			return nil, err
		}

		trs = tox.TokenSource(token)
	}

	mts := vag.MetaTokenSource(func() (*vag.Token, error) {
		var (
			mtoken *vag.Token
			err    error
		)

		if ts := mbb.TokenSource(nil); ts != nil {
			mtoken, err = ts.TokenEx()
		}

		if mtoken == nil || err != nil {
			// get new id token from refreshing TRS token source
			itoken, err := trs.TokenEx()
			if err != nil {
				return nil, err
			}

			// exchange TRS id_token for MBB token
			mtoken, err = mbb.Exchange(url.Values{"id_token": {itoken.IDToken}})
			if err != nil {
				return nil, err
			}
		}

		return mtoken, nil

		// produce tokens from refresh MBB token source
	}, mbb.TokenSource)

	return mts, nil
}
