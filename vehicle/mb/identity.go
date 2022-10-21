package mb

import (
	"context"
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
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.smart-eq

const OAuthURI = "https://id.mercedes-benz.com"

type Identity struct {
	*request.Helper
	oc *oauth2.Config
	oauth2.TokenSource
}

// NewIdentity creates Mercedes Benz identity
func NewIdentity(log *util.Logger, oc *oauth2.Config) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		oc:     oc,
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

	uri := v.oc.AuthCodeURL(state(), oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

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
			if err = v.DoJSON(req, &res); err != nil && len(res.Errors) > 0 {
				err = fmt.Errorf("%s: %w", string(res.Errors[0].Key), err)
			}
		}
	}

	if err == nil {
		data.Password = password
		data.RememberMe = true

		uri = fmt.Sprintf("%s/ciam/auth/login/pass", OAuthURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			if err = v.DoJSON(req, &res); err != nil && len(res.Errors) > 0 {
				err = fmt.Errorf("%s: %w", string(res.Errors[0].Key), err)
			}
		}
	}

	if err == nil && res.Token == "" && res.Result != "" {
		err = fmt.Errorf("missing token: %s", res.Result)
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

	var token *oauth2.Token
	if err == nil {
		ctx, cancel := context.WithTimeout(
			context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
			request.Timeout)
		defer cancel()

		token, err = v.oc.Exchange(ctx, code,
			oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
		)
	}

	if err == nil {
		v.TokenSource = v.oc.TokenSource(context.Background(), token)
	}

	return err
}
