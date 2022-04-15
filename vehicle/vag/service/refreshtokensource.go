package service

import (
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

type RefreshTokenProvider func() (*vag.Token, error)

func DefaultRefreshToken() (*vag.Token, error) {
	return &vag.Token{
		Token: oauth2.Token{
			RefreshToken: refreshToken,
		},
	}, nil
}

// RefreshTokenSource creates a refreshing VAG token source
func RefreshTokenSource(log *util.Logger, tox vag.TokenExchanger, rtp RefreshTokenProvider, q url.Values, user, password string) (vag.TokenSource, error) {
	// create token source from stored refresh token
	if rtp != nil {
		if token, err := rtp(); err == nil {
			trs := tox.TokenSource(token)
			if itoken, err := trs.TokenEx(); err == nil {
				return tox.TokenSource(itoken), nil
			}
		}
	}

	// create token source from fresh login
	q, err := vwidentity.Login(log, q, user, password)
	if err != nil {
		return nil, err
	}

	token, err := tox.Exchange(q)
	if err != nil {
		return nil, err
	}

	return tox.TokenSource(token), nil
}
