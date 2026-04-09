package polestar

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI    = "https://polestarid.eu.polestar.com"
	ClientID    = "l3oopkc_10"
	RedirectURI = "https://www.polestar.com/sign-in-callback"
)

var OAuth2Config = &oauth2.Config{
	ClientID:    ClientID,
	RedirectURL: RedirectURI,
	Endpoint: oauth2.Endpoint{
		AuthURL:   OAuthURI + "/as/authorization.oauth2",
		TokenURL:  OAuthURI + "/as/token.oauth2",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"openid", "profile", "email"},
}

type Identity struct {
	*request.Helper
	user, password string
}

func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	token, err := v.login()
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	return oauth2.ReuseTokenSource(token, OAuth2Config.TokenSource(ctx, token)), nil
}

func (v *Identity) login() (*oauth2.Token, error) {
	v.Client.Jar, _ = cookiejar.New(nil)

	cv := oauth2.GenerateVerifier()

	uri := OAuth2Config.AuthCodeURL(lo.RandomString(16, lo.AlphanumericCharset), oauth2.S256ChallengeOption(cv))
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;",
	})

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	matches := regexp.MustCompile(`(?:url|action):\s*"(.+?)"`).FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return nil, errors.New("could not find resume path")
	}

	resumePath := matches[1]
	if !strings.HasPrefix(resumePath, "http") {
		resumePath = OAuthURI + "/" + strings.TrimLeft(resumePath, "/")
	}

	data := url.Values{
		"pf.username": {v.user},
		"pf.pass":     {v.password},
		"client_id":   {ClientID},
	}

	req, _ = request.New(http.MethodPost, resumePath, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	})

	resp, err = v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	code := resp.Request.URL.Query().Get("code")
	if code == "" {
		return nil, errors.New("missing authorization code")
	}

	cctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ctx, cancel := context.WithTimeout(cctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
