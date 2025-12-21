package zaptec

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
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

// tokenSourceCache stores per-user token sources
var (
	tokenSourceMu    sync.Mutex
	tokenSourceCache = make(map[string]oauth2.TokenSource)

	oidcProvider     *oidc.Provider
	oidcProviderOnce sync.Once
	oidcProviderErr  error
)

// cacheKey generates a unique cache key from user credentials
func cacheKey(user, pass string) string {
	h := sha256.New()
	h.Write([]byte(user))
	h.Write([]byte(":"))
	h.Write([]byte(pass))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ClearTokenCache removes the cached token source for the given user credentials.
// This should be called when credentials change or when a charger is reconfigured.
func ClearTokenCache(user, pass string) {
	tokenSourceMu.Lock()
	defer tokenSourceMu.Unlock()

	key := cacheKey(user, pass)
	delete(tokenSourceCache, key)
}

// getOIDCProvider returns the cached OIDC provider, initializing it once if needed
func getOIDCProvider(ctx context.Context) (*oidc.Provider, error) {
	oidcProviderOnce.Do(func() {
		oidcProvider, oidcProviderErr = oidc.NewProvider(ctx, ApiURL+"/")
	})
	return oidcProvider, oidcProviderErr
}

// GetTokenSource returns a shared oauth2.TokenSource for the given user credentials.
// Multiple chargers using the same user credentials will share the same TokenSource,
// ensuring tokens are reused and authentication is deduplicated.
func GetTokenSource(ctx context.Context, user, pass string) (oauth2.TokenSource, error) {
	tokenSourceMu.Lock()
	defer tokenSourceMu.Unlock()

	// Use hash of username+password as the cache key
	key := cacheKey(user, pass)
	if ts, exists := tokenSourceCache[key]; exists {
		return ts, nil
	}

	// Get the cached OIDC provider (initialized once)
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

	// Wrap with ReuseTokenSource to cache tokens
	ts := oauth2.ReuseTokenSource(token, pts)
	tokenSourceCache[key] = ts

	return ts, nil
}
