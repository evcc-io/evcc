package saic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	TokenSource oauth2.TokenSource
	User        string
	Password    string
	deviceId    string
	baseUrl     string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger, user, password, baseUrl string) *Identity {
	v := &Identity{
		Helper:   request.NewHelper(log),
		User:     user,
		Password: requests.Sha1(password),
		baseUrl:  baseUrl,
	}

	v.deviceId = lo.RandomString(64, lo.AlphanumericCharset) + "###com.saicmotor.europecar"

	v.Client.Transport = &transport.Decorator{
		Decorator: requests.Decorate,
		Base:      v.Client.Transport,
	}

	return v
}

func (v *Identity) Login() error {
	data := url.Values{
		"username":   {v.User},
		"password":   {v.Password}, // Shold be Sha1 encoded
		"grant_type": {"password"},
	}

	token, err := v.retrieveToken(data)
	if err != nil {
		return err
	}

	v.TokenSource = oauth2.ReuseTokenSourceWithExpiry(token, oauth.RefreshTokenSource(token, v.refreshToken), 15*time.Minute)

	return nil
}

func (v *Identity) retrieveToken(data url.Values) (*oauth2.Token, error) {
	data.Set("deviceId", v.deviceId)
	data.Set("response_type", "code")
	data.Set("scope", "all")
	data.Set("deviceType", "0")
	data.Set("loginType", "2")
	data.Set("language", "en")

	// get charging status of vehicle
	req, err := requests.CreateRequest(
		v.baseUrl,
		"oauth/token",
		http.MethodPost,
		data.Encode(),
		request.FormContent,
		"",
		"")
	if err != nil {
		return nil, err
	}

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := requests.DecodeResponse(resp)
	if err != nil {
		return nil, err
	}

	var res requests.Answer[requests.LoginData]
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("%d: %s", res.Code, res.Message)
	}

	tok := oauth2.Token{
		AccessToken:  res.Data.Access_token,
		RefreshToken: res.Data.Refresh_token,
		TokenType:    res.Data.Token_type,
		Expiry:       time.Now().Add(time.Second * time.Duration(res.Data.Expires_in)),
	}

	return &tok, nil
}

func (v *Identity) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"refresh_token": {token.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	token, err := v.retrieveToken(data)
	if token == nil || err != nil {
		// Refresh failed. Try a full login...
		data.Del("refresh_token")
		data.Set("username", v.User)
		data.Set("password", v.Password) // Shold be Sha1 encoded
		data.Set("grant_type", "password")

		return v.retrieveToken(data)
	}

	return token, err
}

func (v *Identity) Token() (*oauth2.Token, error) {
	return v.TokenSource.Token()
}
