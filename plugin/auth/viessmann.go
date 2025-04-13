package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI    = "https://iam.viessmann.com/idp/v3"
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
		Scopes:      []string{"IoT User", "offline_access"},
	}
)

func init() {
	registry.AddCtx("viessmann", NewViessmannFromConfig)
}

type Viessmann struct {
	*request.Helper
	ts oauth2.TokenSource
}

func NewViessmannFromConfig(ctx context.Context, other map[string]any) (Authorizer, error) {
	var cc struct {
		User, Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Viessmann{
		Helper: request.NewHelper(util.NewLogger("viessmann")),
	}

	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)

	token, err := v.login(ctx, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	v.ts = OAuth2Config.TokenSource(ctx, token)

	return v, nil
}

func (v *Viessmann) Transport(base http.RoundTripper) (http.RoundTripper, error) {
	return &oauth2.Transport{
		Base:   base,
		Source: v.ts,
	}, nil
}

func (v *Viessmann) login(ctx context.Context, user, password string) (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv))

	v.Client.Jar, _ = cookiejar.New(nil)
	v.Client.CheckRedirect = request.DontFollow
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": transport.BasicAuthHeader(user, password),
	})

	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	redirect_location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	code := redirect_location.Query().Get("code")

	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))
}

// func (v *Viessmann) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
// 	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
// 	ts := oauth2.ReuseTokenSource(token, OAuth2Config.TokenSource(ctx, token))

// 	token, err := ts.Token()
// 	if err != nil {
// 		token, err = v.login("x", "x")
// 	}

// 	return util.TokenWithExpiry(token), err
// }
