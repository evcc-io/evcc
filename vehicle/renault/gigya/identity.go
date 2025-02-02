package gigya

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

type gigyaResponse struct {
	ErrorCode    int              `json:"errorCode"`    // /accounts.login
	ErrorMessage string           `json:"errorMessage"` // /accounts.login
	SessionInfo  gigyaSessionInfo `json:"sessionInfo"`  // /accounts.login
	IDToken      string           `json:"id_token"`     // /accounts.getJWT
	Data         gigyaData        `json:"data"`         // /accounts.getAccountInfo
}

type gigyaSessionInfo struct {
	CookieValue string `json:"cookieValue"`
}

type gigyaData struct {
	PersonID string `json:"personId"`
}

type Identity struct {
	*request.Helper
	gigya    keys.ConfigServer
	Token    string
	PersonID string
}

func NewIdentity(log *util.Logger, gigya keys.ConfigServer) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		gigya:  gigya,
	}
}

func (v *Identity) Login(user, password string) error {
	sessionCookie, err := v.sessionCookie(user, password)

	if err == nil {
		v.Token, err = v.jwtToken(sessionCookie)
	}

	if err == nil {
		if v.PersonID, err = v.personID(sessionCookie); v.PersonID == "" {
			err = errors.New("missing personID")
		}
	}

	return err
}

func (v *Identity) sessionCookie(user, password string) (string, error) {
	data := url.Values{
		"loginID":  []string{user},
		"password": []string{password},
		"apiKey":   []string{v.gigya.APIKey},
	}

	var res gigyaResponse
	uri := fmt.Sprintf("%s/accounts.login?%s", v.gigya.Target, data.Encode())

	err := v.GetJSON(uri, &res)
	if err == nil && res.ErrorCode > 0 {
		err = errors.New(res.ErrorMessage)
	}

	return res.SessionInfo.CookieValue, err
}

func (v *Identity) personID(sessionCookie string) (string, error) {
	data := url.Values{
		"apiKey":      []string{v.gigya.APIKey},
		"login_token": []string{sessionCookie},
	}

	var res gigyaResponse
	uri := fmt.Sprintf("%s/accounts.getAccountInfo?%s", v.gigya.Target, data.Encode())
	err := v.GetJSON(uri, &res)

	return res.Data.PersonID, err
}

func (v *Identity) jwtToken(sessionCookie string) (string, error) {
	data := url.Values{
		"apiKey":      []string{v.gigya.APIKey},
		"login_token": []string{sessionCookie},
		"fields":      []string{"data.personId,data.gigyaDataCenter"},
		"expiration":  []string{"900"},
	}

	var res gigyaResponse
	uri := fmt.Sprintf("%s/accounts.getJWT?%s", v.gigya.Target, data.Encode())
	err := v.GetJSON(uri, &res)

	return res.IDToken, err
}
