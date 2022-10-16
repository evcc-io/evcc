package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag/loginapps"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

func LoginAppsServiceTokenSource(log *util.Logger, tox *loginapps.Service, loginUrl string, q url.Values, user, password string) (oauth2.TokenSource, error) {
	trs := tox.TokenSource(nil)
	if trs == nil {
		q, err := vwidentity.LoginWithAuthURL(log, loginUrl, q, user, password)
		if err != nil {
			return nil, err
		}

		token, err := tox.Exchange(q)
		if err != nil {
			return nil, err
		}

		trs = tox.TokenSource(token)
	}

	return trs, nil
}
