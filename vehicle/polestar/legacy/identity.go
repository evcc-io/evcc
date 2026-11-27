package legacy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI    = "https://polestarid.eu.polestar.com"
	ClientID    = "lp8dyrd_10" // Polestar app
	RedirectURI = "polestar-explore://explore.polestar.com"
)

var OAuth2Config = &oauth2.Config{
	ClientID:    ClientID,
	RedirectURL: RedirectURI,
	Endpoint: oauth2.Endpoint{
		AuthURL:   OAuthURI + "/as/authorization.oauth2",
		TokenURL:  OAuthURI + "/as/token.oauth2",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"openid", "profile", "email", "customer:attributes", "customer:attributes:write"},
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
	return oauth2.ReuseTokenSource(token, oauth.Redacted(log, OAuth2Config.TokenSource(ctx, token))), nil
}

var (
	loginActionRe = regexp.MustCompile(`"loginAction":\s*"([^"]+)"`)
	pageIdRe      = regexp.MustCompile(`"pageId":\s*"([^"]+)"`)
)

// loginAction returns the form target of the Keycloak login page Polestar ID redirects to
func loginAction(body string) (string, error) {
	m := loginActionRe.FindStringSubmatch(body)
	if m == nil {
		return "", errors.New("could not find login action")
	}

	action := strings.ReplaceAll(m[1], `\/`, "/")

	// credentials must only go to Polestar
	if u, err := url.Parse(action); err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".polestar.com") {
		return "", errors.New("unexpected login action")
	}

	return action, nil
}

// loginError explains why the login page did not redirect back with a code
func loginError(body string) error {
	if strings.Contains(body, "Invalid username or password") {
		return errors.New("invalid username or password")
	}
	if m := pageIdRe.FindStringSubmatch(body); m != nil && m[1] != "login" {
		return fmt.Errorf("additional login step required (%s): sign in once at polestar.com, then retry", m[1])
	}
	return errors.New("missing authorization code")
}

func (v *Identity) login() (*oauth2.Token, error) {
	v.Client.Jar, _ = cookiejar.New(nil)

	// the app's redirect uri carries the code and can't be fetched
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !strings.HasPrefix(req.URL.Scheme, "http") || len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}

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

	action, err := loginAction(string(body))
	if err != nil {
		return nil, err
	}

	data := url.Values{
		"username":     {v.user},
		"password":     {v.password},
		"credentialId": {""},
	}

	req, _ = request.New(http.MethodPost, action, strings.NewReader(data.Encode()), request.URLEncoding)

	resp, err = v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var code string
	if loc, err := resp.Location(); err == nil {
		code = loc.Query().Get("code")
	}
	if code == "" {
		body, _ := io.ReadAll(resp.Body)
		return nil, loginError(string(body))
	}

	cctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ctx, cancel := context.WithTimeout(cctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
