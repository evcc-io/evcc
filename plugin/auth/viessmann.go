package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI    = "https://iam.viessmann-climatesolutions.com/idp/v3"
	RedirectURI = "http://localhost:4200/"
	// ^ the value of RedirectURI doesn't matter, but it must be the same between requests
)

func OAuth2Config(clientID string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		RedirectURL: RedirectURI,
		Scopes:      []string{"IoT User", "offline_access"},
	}
}

func init() {
	registry.AddCtx("viessmann", NewViessmannFromConfig)
}

type Viessmann struct {
	client *http.Client
	ts     oauth2.TokenSource
}

func NewViessmannFromConfig(ctx context.Context, other map[string]any) (Authorizer, error) {
	var cc struct {
		ClientID       string
		User, Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Viessmann{
		client: request.NewClient(util.NewLogger("viessmann")),
	}

	oc := OAuth2Config(cc.ClientID)

	ctx = context.WithValue(context.Background(), oauth2.HTTPClient, v.client)
	token, err := v.login(ctx, oc, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	v.ts = oc.TokenSource(ctx, token)

	return v, nil
}

func (v *Viessmann) Transport(base http.RoundTripper) http.RoundTripper {
	return &oauth2.Transport{
		Base:   base,
		Source: v.ts,
	}
}

func (v *Viessmann) login(ctx context.Context, oc *oauth2.Config, user, password string) (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := oc.AuthCodeURL(state, oauth2.S256ChallengeOption(cv))

	v.client.Jar, _ = cookiejar.New(nil)
	v.client.CheckRedirect = request.DontFollow
	defer func() {
		v.client.Jar = nil
		v.client.CheckRedirect = nil
	}()

	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": transport.BasicAuthHeader(user, password),
	})

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	code := loc.Query().Get("code")

	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
