package mb

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/context"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://id.mercedes-benz.com/.well-known/openid-configuration
const OAuthURI = "https://id.mercedes-benz.com"

var OAuth2Config = &oauth2.Config{
	ClientID:    "70d89501-938c-4bec-82d0-6abb550b0825",
	RedirectURL: "https://oneapp.microservice.smart.com",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://id.mercedes-benz.com/as/authorization.oauth2",
		TokenURL: "https://id.mercedes-benz.com/as/token.oauth2",
	},
	Scopes: []string{"openid", "profile", "email", "phone", "ciam-uid", "offline_access"},
}

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

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func (v *Identity) Login_old(user, password string) error {
	v.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	cv, _ := cv.CreateCodeVerifier()

	params := url.Values{
		"client_id":             {"70d89501-938c-4bec-82d0-6abb550b0825"},
		"redirect_uri":          {"https://oneapp.microservice.smart.com"},
		"response_type":         {"code"},
		"scope":                 {"openid profile email phone ciam-uid offline_access"},
		"code_challenge":        {cv.CodeChallengeS256()},
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

func (v *Identity) Login(user, password string) error {
	if v.Client.Jar == nil {
		var err error
		v.Client.Jar, err = cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
		if err != nil {
			return err
		}
	}

	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return err
	}

	uri := OAuth2Config.AuthCodeURL(state(), oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("redirect_uri", "https://oneapp.microservice.smart.com"),
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	//--

	var resume string
	if err == nil {
		var param request.InterceptResult
		v.Client.CheckRedirect, param = request.InterceptRedirect("resume", false)

		if _, err = v.Get(uri); err == nil {
			resume, err = param()
		}

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

	var req *http.Request
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
			if err = v.DoJSON(req, &res); err != nil {
				if res.Errors != nil && len(res.Errors) > 0 {
					err = fmt.Errorf("%s: %w", string(res.Errors[0].Key), err)
				}
			}
		}
	}

	var code string
	if err == nil {
		params := url.Values{
			"token": {res.Token},
		}

		var param request.InterceptResult
		v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)

		uri := OAuthURI + resume
		if _, err = v.Post(uri, request.FormContent, strings.NewReader(params.Encode())); err == nil {
			code, err = param()
		}

		v.Client.CheckRedirect = nil
	}

	//--

	var token *oauth2.Token
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		token, err = OAuth2Config.Exchange(ctx, code,
			oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
		)
	}

	if err == nil {
		v.TokenSource = OAuth2Config.TokenSource(context.Background(), token)
	}

	return err
}
