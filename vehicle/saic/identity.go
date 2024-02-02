package saic

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	deviceId string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
	}

	var deviceId [64]byte
	for i := 0; i < 64; i++ {
		deviceId[i] = byte(rand.Intn(255))
	}
	v.deviceId = hex.EncodeToString(deviceId[:]) + "###com.saicmotor.europecar"

	v.Client.Transport = &transport.Decorator{
		Decorator: requests.Decorate,
		Base:      v.Client.Transport,
	}

	return v
}

func (v *Identity) Login(user, password string) (oauth2.TokenSource, error) {
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

	// get charging status of vehicle
	req, err := requests.CreateRequest(
		requests.BASE_URL_P+"oauth/token",
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

	var body []byte
	body, err = requests.DecryptAnswer(resp)
	if err == nil {
		err = json.Unmarshal(body, &answer)
		if err == nil && answer.Code != 0 {
			err = fmt.Errorf("%d: %s", answer.Code, answer.Message)
		}
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
