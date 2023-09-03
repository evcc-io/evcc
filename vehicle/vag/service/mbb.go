package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/mbb"
)

// MbbTokenSource creates a refreshing token source for use with the MBB api.
// Once the MBB token expires, it is recreated from the token exchanger (either TokenRefreshService or IDK)
func MbbTokenSource(log *util.Logger, trs vag.TokenSource, clientID string) vag.TokenSource {
	mbb := mbb.New(log, clientID)

	return vag.MetaTokenSource(func() (*vag.Token, error) {
		// get TRS token from refreshing TRS token source
		itoken, err := trs.TokenEx()
		if err != nil {
			return nil, err
		}

		// exchange TRS id_token for MBB token
		mtoken, err := mbb.Exchange(url.Values{"id_token": {itoken.IDToken}})
		if err != nil {
			return nil, err
		}

		return mtoken, err

		// produce tokens from refresh MBB token source
	}, mbb.TokenSource)
}
