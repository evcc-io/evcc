package subaru

import (
	"fmt"
	"maps"
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
	APIVersion       = "protocol=1.0,resource=2.1"
	ClientID         = "8c4921b0b08901fef389ce1af49c4e10.subaru.com"
	Scope            = "openid profile vehicles"
	RedirectURI      = "com.subaru.oneapp:/oauth2Callback"
	AppAuthorization = "Basic OGM0OTIxYjBiMDg5MDFmZWYzODljZTFhZjQ5YzRlMTAuc3ViYXJ1LmNvbTpJaGNkcjV4YmhIYlRSMk9aOGdRa3YyNTZicmhTYjc="
)

type Identity struct {
	log *util.Logger
	*request.Helper
	oauth2.TokenSource
	uuid    string
	idToken string
}

func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}
}

func (v *Identity) IDToken() string {
	return v.idToken
}

func (v *Identity) authenticate(initial Auth, user, password string) (*Token, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthenticationPath)

	auth := initial
	passwordSet := false

	for {
		for i, cb := range auth.Callbacks {
			switch cb.Type {
			case "NameCallback":
				if outputValue, ok := cb.Output[0].Value.(string); ok && outputValue == "User Name" {
					auth.Callbacks[i].Input[0].Value = user
				}
			case "PasswordCallback":
				auth.Callbacks[i].Input[0].Value = password
				passwordSet = true
			}
		}

		req, err := request.New(http.MethodPost, uri, request.MarshalJSON(auth), request.JSONEncoding)
		if err != nil {
			return nil, err
		}

		if passwordSet {
			var token Token
			if err := v.DoJSON(req, &token); err != nil {
				return nil, err
			}
			return &token, nil
		}

		var next Auth
		if err := v.DoJSON(req, &next); err != nil {
			return nil, err
		}
		auth = next
	}
}

func (v *Identity) authorize(token Token) (string, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthorizationPath)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Cookie": fmt.Sprintf("iPlanetDirectoryPro=%s", token.TokenID),
	})
	if err != nil {
		return "", err
	}

	originalCheckRedirect := v.Client.CheckRedirect
	defer func() { v.Client.CheckRedirect = originalCheckRedirect }()

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

	headers := make(map[string]string)
	maps.Copy(headers, request.URLEncoding)
	headers["Authorization"] = AppAuthorization
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

	v.idToken = res.IDToken

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
	v.TokenSource = oauth.RefreshTokenSource(util.TokenWithExpiry(&res.Token), v.refreshToken)
	return nil
}

func (v *Identity) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AccessTokenPath)
	data := url.Values{
		"client_id":     {ClientID},
		"redirect_uri":  {RedirectURI},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	headers := make(map[string]string)
	maps.Copy(headers, request.URLEncoding)
	headers["Authorization"] = AppAuthorization
	var res oauth2.Token
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers)
	if err != nil {
		return nil, err
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}
	return util.TokenWithExpiry(&res), nil
}

func (v *Identity) Login(user, password string) error {
	uri := fmt.Sprintf("%s/%s", BaseUrl, AuthenticationPath)
	req, err := request.New(http.MethodPost, uri, nil, map[string]string{
		"Accept":   "application/json",
		"X-Region": "EU",
		"Region":   "EU",
		"Brand":    "S",
		"X-Brand":  "S",
	})
	if err != nil {
		return fmt.Errorf("failed to create authentication request: %w", err)
	}

	var auth Auth
	if err = v.DoJSON(req, &auth); err != nil {
		return fmt.Errorf("failed to get initial auth response: %w", err)
	}

	token, err := v.authenticate(auth, user, password)
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
