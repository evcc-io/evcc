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
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	RedirectURI = "com.bmw.connected://oauth"
)

type Identity struct {
	*request.Helper
	region Region
}

// NewIdentity creates BMW identity
func NewIdentity(log *util.Logger, region string) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
		region: regions[strings.ToUpper(region)],
	}

	return v
}

func (v *Identity) Login(user, password string) (oauth2.TokenSource, error) {
	v.Client.CheckRedirect = request.DontFollow
	defer func() { v.Client.CheckRedirect = nil }()

	cv := oauth2.GenerateVerifier()

	v.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	data := url.Values{
		"client_id":             {v.region.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {RedirectURI},
		"state":                 {v.region.State},
		"scope":                 {"openid profile email offline_access smacc vehicle_data perseus dlm svds cesim vsapi remote_services fupo authenticate_user"},
		"nonce":                 {"login_nonce"},
		"code_challenge_method": {"S256"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(cv)},
		"username":              {user},
		"password":              {password},
		"grant_type":            {"authorization_code"},
	}

	uri := fmt.Sprintf("%s/oauth/authenticate", v.region.AuthURI)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	var res struct {
		RedirectTo       string `json:"redirect_to"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := v.DoJSON(req, &res); err != nil {
		if res.ErrorDescription != "" {
			err = fmt.Errorf("%s: %w", res.ErrorDescription, err)
		}
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
		"code_verifier": {cv},
	}

	token, err := v.retrieveToken(data)
	if err != nil {
		return nil, err
	}

	ts := oauth2.ReuseTokenSourceWithExpiry(token, oauth.RefreshTokenSource(token, v), 15*time.Minute)

	return ts, nil
}

func (v *Identity) retrieveToken(data url.Values) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/oauth/token", v.region.AuthURI)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": v.region.Token.Authorization,
	})

	var tok oauth.Token
	if err == nil {
		err = v.DoJSON(req, &tok)
	}

	return (*oauth2.Token)(&tok), err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"redirect_uri":  []string{RedirectURI},
		"refresh_token": []string{token.RefreshToken},
		"grant_type":    []string{"refresh_token"},
	}

	return v.retrieveToken(data)
}
