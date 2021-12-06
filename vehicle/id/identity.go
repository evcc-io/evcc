package id

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

const (
	// OauthTokenURI is the login uri for ID vehicles
	OauthTokenURI = "https://login.apps.emea.vwapps.io"
)

type Identity struct {
	*request.Helper
	idtp *vw.IDTokenProvider
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger, user, password string) *Identity {
	uri := fmt.Sprintf("%s/authorize?%s", OauthTokenURI, AuthParams.Encode())

	return &Identity{
		Helper: request.NewHelper(log),
		idtp:   vw.NewIDTokenProvider(log, uri, user, password),
	}
}

func (v *Identity) Login() error {
	token, err := v.login()
	if err != nil {
		return err
	}

	v.TokenSource = oauth.RefreshTokenSource((*oauth2.Token)(&token), v)

	return nil
}

func (v *Identity) login() (Token, error) {
	q, err := v.idtp.Login()

	if err == nil {
		for _, k := range []string{"state", "id_token", "access_token", "code"} {
			if q.Get(k) == "" {
				err = errors.New("missing " + k)
				break
			}
		}
	}

	var token Token
	if err == nil {
		data := map[string]string{
			"state":             q.Get("state"),
			"id_token":          q.Get("id_token"),
			"redirect_uri":      "weconnect://authenticated",
			"region":            "emea",
			"access_token":      q.Get("access_token"),
			"authorizationCode": q.Get("code"),
		}

		var req *http.Request
		uri := fmt.Sprintf("%s/login/v1", OauthTokenURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	return token, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/refresh/v1", OauthTokenURI)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + token.RefreshToken,
	})

	var res Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return (*oauth2.Token)(&res), err
}
