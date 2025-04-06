package viessmann

import (
	"context"
	"fmt"
  "os"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	ApiURI = "https://api.viessmann.com/iot/v2"
	OAuthURI = "https://iam.viessmann.com/idp/v3"
  RedirectURI = "http://localhost:4200/"
  // ^ the value of RedirectURI doesn't matter, but it must be the same between requests
)

var (
  User     = os.Getenv("VIESSMANN_USER")
  Password = os.Getenv("VIESSMANN_PASS")
  ClientId = os.Getenv("VIESSMANN_CLIENT_ID")
  // TODO: ^ these should really come from the evcc config...
	OAuth2Config = &oauth2.Config{
		ClientID: ClientId,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		RedirectURL: RedirectURI,
		Scopes: []string{"IoT User", "offline_access"},
	}
)

type Identity struct {
	*request.Helper
	user, password string
}

func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	v := &Identity{
		Helper: request.NewHelper(log),
		user:       user,
		password:   password,
	}

  httpClient := v.Helper
  httpClient.Transport = transport.BasicAuth(user, password, httpClient.Transport)
	token, err := v.login()

	return oauth.RefreshTokenSource(token, v), err
}

func (v *Identity) login() (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv))

	v.Client.Jar, _ = cookiejar.New(nil)
	v.Client.CheckRedirect = request.DontFollow
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	resp, err := v.Client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

  fmt.Println(fmt.Sprintf("XXX0 %d", resp.StatusCode))
	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d - did you provide correct username/password?", resp.StatusCode)
	}

	// username
	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

  fmt.Println(fmt.Sprintf("XXX2 u=%s", u))

	query := u.Query()
	query.Set("username", v.user)
	query.Set("js-available", "true")
	query.Set("webauthn-available", "false")
	query.Set("is-brave", "false")
	query.Set("webauthn-platform-available", "false")
	query.Set("action", "default")

	resp, err = v.PostForm(OAuthURI+u.String(), query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	// password
	u, err = url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	query = u.Query()
	query.Set("username", v.user)
	query.Set("password", v.password)
	query.Set("action", "default")

	resp, err = v.PostForm(OAuthURI+u.String(), query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)

	// resume
	u, err = url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	resp, err = v.Get(OAuthURI + u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	code, err := param()
	if err != nil {
		return nil, err
	}

	cctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ctx, cancel := context.WithTimeout(cctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ts := oauth2.ReuseTokenSource(token, OAuth2Config.TokenSource(ctx, token))

	token, err := ts.Token()
	if err != nil {
		token, err = v.login()
	}

	return token, err
}
