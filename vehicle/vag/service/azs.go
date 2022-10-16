package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/aazsproxy"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
)

// AAZSTokenSource creates a refreshing token source for use with the AAZS api.
// Once the AAZS token expires, it is recreated from the token exchanger (either TokenRefreshService or IDK).
// Return values are the AAZS and token exchanger (TRS or IDK) token sources.
func AAZSTokenSource(log *util.Logger, tox vag.TokenExchanger, azs *aazsproxy.Service, azsConfig string, q url.Values, user, password string) (vag.TokenSource, vag.TokenSource, error) {
	trs := tox.TokenSource(nil)
	if trs == nil {
		q, err := vwidentity.Login(log, q, user, password)
		if err != nil {
			return nil, nil, err
		}

		token, err := tox.Exchange(q)
		if err != nil {
			return nil, nil, err
		}

		trs = tox.TokenSource(token)
	}

	mts := vag.MetaTokenSource(func() (*vag.Token, error) {
		var (
			atoken *vag.Token
			err    error
		)

		if ts := azs.TokenSource(nil); ts != nil {
			atoken, err = ts.TokenEx()
		}

		if atoken == nil || err != nil {
			// get new id token from refreshing TRS token source
			itoken, err := trs.TokenEx()
			if err != nil {
				return nil, err
			}

			// exchange TRS id_token for AAZS token
			atoken, err = azs.Exchange(azsConfig, itoken.IDToken)
			if err != nil {
				return nil, err
			}
		}

		return atoken, nil

		// produce tokens from refresh MBB token source
	}, azs.TokenSource)

	return mts, trs, nil
}
