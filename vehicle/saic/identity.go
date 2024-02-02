package saic

import (
	"encoding/hex"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"golang.org/x/oauth2"
)

type Identity struct {
	deviceId string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{}

	var deviceId [64]byte
	for i := 0; i < 64; i++ {
		deviceId[i] = byte(rand.Intn(255))
	}
	v.deviceId = hex.EncodeToString(deviceId[:]) + "###com.saicmotor.europecar"

	return v
}

func (v *Identity) Login(user, password string) (oauth2.TokenSource, error) {
	// login

	data := url.Values{
		"username":   {user},
		"password":   {requests.Sha1(password)}, // Shold be Sha1 encoded
		"grant_type": {"password"},
	}

	token, err := v.retrieveToken(data)
	if err != nil {
		return nil, err
	}

	ts := oauth2.ReuseTokenSourceWithExpiry(token, oauth.RefreshTokenSource(token, v), 15*time.Minute)

	return ts, nil

}

func (v *Identity) retrieveToken(data url.Values) (*oauth2.Token, error) {

	var loginData requests.LoginData
	answer := requests.Answer{
		Data: &loginData,
	}

	data.Set("deviceId", v.deviceId)
	data.Set("response_type", "code")
	data.Set("scope", "all")
	data.Set("deviceType", "0")
	data.Set("loginType", "2")
	data.Set("language", "en")

	_, err := requests.SendRequest(
		requests.BASE_URL_P+"oauth/token",
		http.MethodPost,
		data.Encode(),
		"application/x-www-form-urlencoded",
		"",
		"",
		&answer)

	if err != nil {
		return nil, err
	}

	tok := oauth2.Token{
		AccessToken:  loginData.Access_token,
		RefreshToken: loginData.Refresh_token,
		TokenType:    loginData.Token_type,
	}

	tok.Expiry = time.Now().Add(time.Second * time.Duration(loginData.Expires_in))

	return &tok, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {

	data := url.Values{
		"refresh_token": []string{token.RefreshToken},
		"grant_type":    []string{"refresh_token"},
	}
	return v.retrieveToken(data)
}
