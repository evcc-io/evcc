package viessmann

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
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

type Identity struct {
	*request.Helper
	user, password string
}

func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	refresher := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	token, err := refresher.login()

	return oauth.RefreshTokenSource(token, refresher), err
}

func (v *Identity) login() (*oauth2.Token, error) {
	code_verifier := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(code_verifier))

	v.Client.Jar, _ = cookiejar.New(nil)
	v.Client.CheckRedirect = request.DontFollow
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	// we need to set basicauth for this and only this request - there's probably an easier way to do this rather than the Transport...
	httpClient := v.Helper
	httpClient.Transport = transport.BasicAuth(v.user, v.password, httpClient.Transport)
	resp, err := v.Client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	httpClient.Transport = nil // see comment above

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	redirect_location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	code := redirect_location.Query().Get("code")

	cctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ctx, cancel := context.WithTimeout(cctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(
		ctx,
		code,
		oauth2.VerifierOption(code_verifier),
		oauth2.SetAuthURLParam("client_id", ClientId),
	)
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
