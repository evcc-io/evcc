package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util/log"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/aazsproxy"
)

// AAZSTokenSource creates a refreshing token source for use with the AAZS api.
// Once the AAZS token expires, it is recreated from the token exchanger (either TokenRefreshService or IDK).
// Return values are the AAZS and token exchanger (TRS or IDK) token sources.
func AAZSTokenSource(log log.Logger, tox vag.TokenExchanger, azsConfig string, q url.Values) (vag.TokenSource, vag.TokenSource, error) {
	token, err := tox.Exchange(q)
	if err != nil {
		return nil, nil, err
	}

	trs := tox.TokenSource(token)
	azs := aazsproxy.New(log)

	mts := vag.MetaTokenSource(func() (*vag.Token, error) {
		// get TRS token from refreshing TRS token source
		itoken, err := trs.TokenEx()
		if err != nil {
			return nil, err
		}

		// exchange TRS id_token for AAZS token
		atoken, err := azs.Exchange(azsConfig, itoken.IDToken)
		if err != nil {
			return nil, err
		}

		return atoken, err

		// produce tokens from refresh MBB token source
	}, azs.TokenSource)

	return mts, trs, nil
}
