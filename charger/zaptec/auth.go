package zaptec

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util/cache"
	"golang.org/x/oauth2"
)

// passwordTokenSource implements oauth2.TokenSource for password grant flow
type passwordTokenSource struct {
	ctx    context.Context
	config *oauth2.Config
	user   string
	pass   string
}

// Token returns a token or an error.
// Implements oauth2.TokenSource interface
func (p *passwordTokenSource) Token() (*oauth2.Token, error) {
	return p.config.PasswordCredentialsToken(p.ctx, p.user, p.pass)
}

var (
	// TokenSourceCache stores per-user token sources
	tokenSourceCache = cache.New[oauth2.TokenSource]()

	oidcProvider     *oidc.Provider
	oidcProviderOnce sync.Once
	oidcProviderErr  error
)

// getOIDCProvider returns the cached OIDC provider, initializing it once if needed
func getOIDCProvider(ctx context.Context) (*oidc.Provider, error) {
	oidcProviderOnce.Do(func() {
		oidcProvider, oidcProviderErr = oidc.NewProvider(ctx, ApiURL+"/")
	})
	return oidcProvider, oidcProviderErr
}

// TokenSource returns a shared oauth2.TokenSource for the given user.
func TokenSource(ctx context.Context, user, pass string) (oauth2.TokenSource, error) {
	return tokenSourceCache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		provider, err := getOIDCProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
		}

		oc := &oauth2.Config{
			Endpoint: provider.Endpoint(),
			Scopes: []string{
				oidc.ScopeOpenID,
			},
		}

		// Create the password token source
		pts := &passwordTokenSource{
			ctx:    ctx,
			config: oc,
			user:   user,
			pass:   pass,
		}

		// Get initial token
		token, err := pts.Token()
		if err != nil {
			return nil, err
		}

		return oauth2.ReuseTokenSource(token, pts), nil
	})
}
