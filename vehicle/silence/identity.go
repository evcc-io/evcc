package silence

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/identitytoolkit/v3"
	"google.golang.org/api/option"
)

// tokenSource is an oauth2.TokenSource
type tokenSource struct {
	identitytoolkitService *identitytoolkit.Service
	user, password         string
	oauth2.TokenSource
}

// TokenSource creates an STS token source
func TokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	ctx := context.Background()
	helper := request.NewHelper(log)

	identitytoolkitService, err := identitytoolkit.NewService(ctx, option.WithHTTPClient(helper.Client))
	if err != nil {
		return nil, err
	}

	c := &tokenSource{
		identitytoolkitService: identitytoolkitService,
		user:                   user,
		password:               password,
	}

	token, err := c.Login()
	if err == nil {
		c.TokenSource = oauth.RefreshTokenSource(token, c)
	}

	return c, err
}

func (c *tokenSource) Login() (*oauth2.Token, error) {
	req := &identitytoolkit.IdentitytoolkitRelyingpartyVerifyPasswordRequest{
		Email:             c.user,
		Password:          c.password,
		ReturnSecureToken: true,
	}

	call := c.identitytoolkitService.Relyingparty.VerifyPassword(req)

	resp, err := call.Do(googleapi.QueryParameter("key", ApiKey))
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  resp.IdToken,
		RefreshToken: resp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}

	return token, nil
}

func (c *tokenSource) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	return c.Login()
}
