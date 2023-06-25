package bmw

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	AuthURI     = "https://customer.bmwgroup.com/gcdm/oauth"
	RedirectURI = "com.bmw.connected://oauth"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	user, password string
}

// NewIdentity creates BMW identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
	}

	return v
}

func (v *Identity) Login(user, password string) error {
	v.user = user
	v.password = password

	token, err := v.RefreshToken(nil)

	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource(token, v)
	}

	return err
}

func (v *Identity) retrieveToken(data url.Values) (*oauth2.Token, error) {
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	uri := fmt.Sprintf("%s/token", AuthURI)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": "Basic MzFjMzU3YTAtN2ExZC00NTkwLWFhOTktMzNiOTcyNDRkMDQ4OmMwZTMzOTNkLTcwYTItNGY2Zi05ZDNjLTg1MzBhZjY0ZDU1Mg==",
	})

	if err == nil {
		err = v.DoJSON(req, &tok)
	}

	token := &oauth2.Token{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}

	return token, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token == nil || token.RefreshToken == "" {
		return v.login()
	}

	data := url.Values{
		"redirect_uri":  []string{RedirectURI},
		"refresh_token": []string{token.RefreshToken},
		"grant_type":    []string{"refresh_token"},
	}

	return v.retrieveToken(data)
}

func (v *Identity) login() (*oauth2.Token, error) {
	v.Client.CheckRedirect = request.DontFollow
	defer func() { v.Client.CheckRedirect = nil }()

	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, err
	}

	v.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	data := url.Values{
		"client_id":             {"31c357a0-7a1d-4590-aa99-33b97244d048"},
		"response_type":         {"code"},
		"redirect_uri":          {RedirectURI},
		"state":                 {"cwU-gIE27j67poy2UcL3KQ"},
		"scope":                 {"openid profile email offline_access smacc vehicle_data perseus dlm svds cesim vsapi remote_services fupo authenticate_user"},
		"nonce":                 {"login_nonce"},
		"code_challenge_method": {"S256"},
		"code_challenge":        {cv.CodeChallengeS256()},
		"username":              {v.user},
		"password":              {v.password},
		"grant_type":            {"authorization_code"},
	}

	uri := fmt.Sprintf("%s/authenticate", AuthURI)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	var res struct {
		RedirectTo string `json:"redirect_to"`
	}

	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	query, err := url.ParseQuery(strings.TrimPrefix(res.RedirectTo, "redirect_uri=com.bmw.connected://oauth?"))
	if err != nil {
		return nil, err
	}

	auth := query.Get("authorization")
	if auth == "" {
		return nil, errors.New("authorization code not found")
	}

	data.Set("authorization", auth)
	delete(data, "username")
	delete(data, "password")
	delete(data, "grant_type")

	req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	uri = resp.Header.Get("Location")
	if uri == "" {
		return nil, errors.New("authorization code not found")
	}

	query, err = url.ParseQuery(strings.TrimPrefix(uri, "com.bmw.connected://oauth?"))
	if err != nil {
		return nil, err
	}

	data = url.Values{
		"code":          {query.Get("code")},
		"redirect_uri":  {RedirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {cv.CodeChallengePlain()},
	}

	return v.retrieveToken(data)
}
