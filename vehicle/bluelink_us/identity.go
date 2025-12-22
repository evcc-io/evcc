package bluelink_us

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	BaseURL      = "https://api.telematics.hyundaiusa.com"
	TokenURL     = "/v2/ac/oauth/token"
	ClientID     = "m66129Bb-em93-SPAHYN-bZ91-am4540zp19920"
	ClientSecret = "v558o935-6nne-423i-baa8"
)

// BaseHeaders returns the common headers required for all API requests
func BaseHeaders() map[string]string {
	// Calculate UTC offset
	_, offset := time.Now().Zone()
	offsetHours := fmt.Sprintf("%d", offset/3600)

	return map[string]string{
		"content-type":   "application/json;charset=UTF-8",
		"accept":         "application/json, text/plain, */*",
		"from":           "SPA",
		"to":             "ISS",
		"language":       "0",
		"offset":         offsetHours,
		"brandIndicator": "H",
		"client_id":      ClientID,
		"clientSecret":   ClientSecret,
		"refresh":        "false",
	}
}

type Identity struct {
	*request.Helper
	user, password string
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

func (v *Identity) Login() error {
	if v.user == "" || v.password == "" {
		return api.ErrMissingCredentials
	}

	token, err := v.login()
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	v.TokenSource = oauth.RefreshTokenSource(token, v)
	return nil
}

func (v *Identity) login() (*oauth2.Token, error) {
	data := LoginRequest{
		Username: v.user,
		Password: v.password,
	}

	req, err := request.New(http.MethodPost, BaseURL+TokenURL, request.MarshalJSON(data), BaseHeaders())
	if err != nil {
		return nil, err
	}

	var res TokenResponse
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	// Parse expires_in from string
	expiresIn, _ := strconv.Atoi(res.ExpiresIn)
	if expiresIn == 0 {
		expiresIn = 3600 // default to 1 hour
	}

	return &oauth2.Token{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

// RefreshToken implements oauth.TokenRefresher
// US API doesn't have a refresh endpoint, so we re-login with credentials
func (v *Identity) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	return v.login()
}
