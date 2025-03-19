package toyota

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const (
	APIVersion   = "protocol=1.0,resource=2.1"
	ClientID     = "oneapp"
	ClientSecret = "6GKIax7fGT5yPHuNmWNVOc4q5POBw1WRSW39ubRA8WPBmQ7MOxhm75EsmKMKENem"
	Scope        = "openid profile vehicles"
	Realm        = "a-ncb-prod"
	RedirectURI  = "com.toyota.oneapp:/oauth2Callback"
)

type Identity struct {
	log *util.Logger
	*request.Helper
	oauth2.TokenSource
	uuid string
}

func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}
}

func (v *Identity) authenticate(auth Auth, user, password string, passwordSet bool) (*Token, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthenticationPath)

	// Update callbacks with credentials
	for id, cb := range auth.Callbacks {
		switch cb.Type {
		case "NameCallback":
			outputValue, ok := cb.Output[0].Value.(string)
			if ok && outputValue == "User Name" {
				auth.Callbacks[id].Input[0].Value = user
			}
		case "PasswordCallback":
			auth.Callbacks[id].Input[0].Value = password
			passwordSet = true
		}
	}

	// Send authentication request
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(auth), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	// If we've already set the password, expect a token response
	if passwordSet {
		var token Token
		if err := v.DoJSON(req, &token); err != nil {
			return nil, err
		}
		return &token, nil
	}

	// Otherwise continue with Auth flow
	var res Auth
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	// Continue authentication flow
	return v.authenticate(res, user, password, passwordSet)
}

func (v *Identity) authorize(token Token) (string, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthorizationPath)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Cookie": fmt.Sprintf("iPlanetDirectoryPro=%s", token.TokenID),
	})
	if err != nil {
		return "", err
	}
	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)
	var code string
	if _, err = v.Do(req); err == nil {
		code, err = param()
	}
	return code, err
}

func (v *Identity) fetchTokenCredentials(code string) error {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AccessTokenPath)

	data := url.Values{
		"client_id":     {ClientID},
		"code":          {code},
		"redirect_uri":  {RedirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {"plain"},
	}

	headers := request.URLEncoding
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers)
	if err != nil {
		return err
	}

	var res struct {
		oauth2.Token
		IDToken string `json:"id_token"`
	}
	if err = v.DoJSON(req, &res); err != nil {
		return fmt.Errorf("failed to fetch token credentials: %w", err)
	}

	// Parse ID token without verification to extract UUID
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(res.IDToken, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse id token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims format")
	}

	uuid, ok := claims["uuid"].(string)
	if !ok {
		return fmt.Errorf("uuid claim missing or invalid in token")
	}

	v.uuid = uuid
	v.TokenSource = oauth.RefreshTokenSource(util.TokenWithExpiry(&res.Token), v)
	return nil
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AccessTokenPath)
	data := url.Values{
		"client_id":     {ClientID},
		"redirect_uri":  {RedirectURI},
		"grant_type":    {"refresh_token"},
		"code_verifier": {"plain"},
		"refresh_token": {token.RefreshToken},
	}
	var res oauth2.Token
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}
	return util.TokenWithExpiry(&res), nil
}

func (v *Identity) Login(user, password string) error {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthenticationPath)
	req, err := request.New(http.MethodPost, uri, nil, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to create authentication request: %w", err)
	}

	var auth Auth
	if err = v.DoJSON(req, &auth); err != nil {
		return fmt.Errorf("failed to get initial auth response: %w", err)
	}

	token, err := v.authenticate(auth, user, password, false)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("token error: %w", err)
	}

	code, err := v.authorize(*token)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	if err = v.fetchTokenCredentials(code); err != nil {
		return fmt.Errorf("failed to fetch token credentials: %w", err)
	}

	return nil
}
