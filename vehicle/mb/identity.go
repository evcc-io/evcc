package mb

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://id.mercedes-benz.com/.well-known/openid-configuration
const OAuthURI = "https://id.mercedes-benz.com"

type Identity struct {
	*request.Helper
	oauth2.TokenSource
}

// NewIdentity creates Mercedes Benz identity
func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
	}
}

func (v *Identity) Login(user, password string) error {
	v.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	cv, _ := cv.CreateCodeVerifier()
	codeChallenge := cv.CodeChallengeS256()

	params := url.Values{
		"client_id":             {"70d89501-938c-4bec-82d0-6abb550b0825"},
		"redirect_uri":          {"https://oneapp.microservice.smart.com"},
		"response_type":         {"code"},
		"scope":                 {"openid profile email phone ciam-uid offline_access"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	uri := fmt.Sprintf("%s/as/authorization.oauth2", OAuthURI)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		req.URL.RawQuery = params.Encode()
	}

	var resume string
	if err == nil {
		v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if resume == "" {
				resume = req.URL.Query().Get("resume")
			}
			return nil
		}
		_, err = v.Do(req)
		v.Client.CheckRedirect = nil
	}

	data := struct {
		Username   string `json:"username"`
		Password   string `json:"password,omitempty"`
		RememberMe bool   `json:"rememberMe,omitempty"`
	}{
		Username: user,
	}

	var res struct {
		Result, Token string
		Errors        []struct{ Key string }
	}
	if err == nil {
		uri = fmt.Sprintf("%s/ciam/auth/login/user", OAuthURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			err = v.DoJSON(req, &res)
		}
	}

	if err == nil {
		data.Password = password
		data.RememberMe = true

		uri = fmt.Sprintf("%s/ciam/auth/login/pass", OAuthURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			if err := v.DoJSON(req, &res); err != nil {
				if res.Errors != nil && len(res.Errors) > 0 {
					err = fmt.Errorf("%s: %w", string(res.Errors[0].Key), err)
				}
				return err
			}
		}
	}

	var code string
	if err == nil {
		params = url.Values{
			"token": {res.Token},
		}

		uri := OAuthURI + resume
		req, err = request.New(http.MethodPost, uri, strings.NewReader(params.Encode()), request.URLEncoding)
		if err == nil {
			v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				if code == "" {
					if code = req.URL.Query().Get("code"); code != "" {
						return http.ErrUseLastResponse
					}
				}
				return nil
			}
			_, err = v.Do(req)
			v.Client.CheckRedirect = nil
		}
	}

	var token oauth2.Token
	if err == nil {
		params = url.Values{
			"client_id":     {"70d89501-938c-4bec-82d0-6abb550b0825"},
			"redirect_uri":  {"https://oneapp.microservice.smart.com"},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"code_verifier": {cv.CodeChallengePlain()},
		}

		uri = fmt.Sprintf("%s/as/token.oauth2", OAuthURI)

		req, err = request.New(http.MethodPost, uri, strings.NewReader(params.Encode()), request.URLEncoding)
		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	if err == nil {
		v.TokenSource = oauth2.StaticTokenSource(&token)
	}

	return err
}
