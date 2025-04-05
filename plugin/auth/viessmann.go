package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

type Viessmann struct {
	*http.Client
	ts oauth2.TokenSource
}

func init() {
	registry.AddCtx("viessmann", NewViessmannFromConfig)
}

func NewViessmannFromConfig(ctx context.Context, other map[string]any) (Authorizer, error) {
	var cc struct {
		User, Password string
		ClientID       string
		RedirectURL    string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Viessmann{
		Client: request.NewClient(util.NewLogger("viessmann")),
	}

	oc := oauth2.Config{
		ClientID: cc.ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://iam.viessmann.com/idp/v3/authorize",
		},
		RedirectURL: cc.RedirectURL,
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, v.Client)

	token, err := v.login(ctx, oc, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	v.ts = oc.TokenSource(ctx, token)

	return v, nil
}

func (v *Viessmann) login(ctx context.Context, oc oauth2.Config, user, password string) (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := oc.AuthCodeURL(state, oauth2.S256ChallengeOption(cv),
		oauth2.SetAuthURLParam("user", transport.BasicAuthHeader(user, password)),
	)

	v.Client.CheckRedirect = request.DontFollow

	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": transport.BasicAuthHeader(user, password),
	})

	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	v.Client.CheckRedirect = nil

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	// username
	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	code := u.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("code not found")
	}

	token, err := oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (v *Viessmann) Transport(base http.RoundTripper) (http.RoundTripper, error) {
	return &oauth2.Transport{
		Source: v.ts,
		Base:   base,
	}, nil
}
