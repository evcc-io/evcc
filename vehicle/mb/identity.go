package mb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.smart-eq

// https://id.mercedes-benz.com/.well-known/openid-configuration
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

func (v *Identity) Login(user, password string) error {
	if v.Client.Jar == nil {
		v.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
	}

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("resume", false)

	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := v.oc.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(cv))
	if _, err := v.Get(uri); err != nil {
		return err
	}

	resume, err := param()
	if err != nil {
		return err
	}

	v.Client.CheckRedirect = nil

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

	uri = fmt.Sprintf("%s/ciam/auth/login/user", OAuthURI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		if err = v.DoJSON(req, &res); err != nil && len(res.Errors) > 0 {
			err = fmt.Errorf("%s: %w", res.Errors[0].Key, err)
		}
	}

	if err == nil {
		data.Password = password
		data.RememberMe = true

		uri = fmt.Sprintf("%s/ciam/auth/login/pass", OAuthURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			if err = v.DoJSON(req, &res); err != nil && len(res.Errors) > 0 {
				err = fmt.Errorf("%s: %w", res.Errors[0].Key, err)
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

		token, err = v.oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
	}

	if err == nil {
		v.TokenSource = v.oc.TokenSource(context.Background(), token)
	}

	return err
}
