package tesla

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/uhthomas/tesla"
	"golang.org/x/oauth2"
)

// Client is the tesla authentication client
type Client struct {
	config   *oauth2.Config
	auth     *tesla.Auth
	verifier string
}

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// https://www.oauth.com/oauth2-servers/pkce/
func pkce() (verifier, challenge string, err error) {
	var p [87]byte
	if _, err := io.ReadFull(rand.Reader, p[:]); err != nil {
		return "", "", fmt.Errorf("rand read full: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(p[:])
	b := sha256.Sum256([]byte(challenge))
	challenge = base64.RawURLEncoding.EncodeToString(b[:])
	return verifier, challenge, nil
}

// NewClient creates a tesla authentication client
func NewClient(log *util.Logger) (*Client, error) {
	httpClient := &http.Client{Transport: &roundTripper{
		log:       log,
		transport: http.DefaultTransport,
	}}

	config := &oauth2.Config{
		ClientID:     "ownerapi",
		ClientSecret: "",
		RedirectURL:  "https://auth.tesla.com/void/callback",
		Scopes:       []string{"openid email offline_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.tesla.com/oauth2/v3/authorize",
			TokenURL: "https://auth.tesla.com/oauth2/v3/token",
		},
	}

	verifier, challenge, err := pkce()
	if err != nil {
		return nil, fmt.Errorf("pkce: %w", err)
	}

	auth := &tesla.Auth{
		Client: httpClient,
		AuthURL: config.AuthCodeURL(state(), oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("code_challenge", challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		),
	}

	client := &Client{
		config:   config,
		auth:     auth,
		verifier: verifier,
	}
	client.DeviceHandler(client.mfaUnsupported)

	return client, nil
}

// Login executes the MFA or non-MFA login
func (c *Client) Login(username, password string) (oauth2.TokenSource, error) {
	ctx := context.Background()
	code, err := c.auth.Do(ctx, username, password)
	if err != nil {
		return nil, err
	}

	token, err := c.config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", c.verifier),
	)
	if err != nil {
		return nil, fmt.Errorf("exchange: %w", err)
	}

	return c.TokenSource(token), nil
}

// TokenSource creates an oauth tokensource from given token
func (c *Client) TokenSource(token *oauth2.Token) oauth2.TokenSource {
	ctx := context.Background()
	return c.config.TokenSource(ctx, token)
}

// DeviceHandler sets an alternative authentication device handler
func (c *Client) DeviceHandler(handler func(context.Context, []tesla.Device) (tesla.Device, string, error)) {
	c.auth.SelectDevice = handler
}

func (c *Client) mfaUnsupported(_ context.Context, _ []tesla.Device) (tesla.Device, string, error) {
	return tesla.Device{}, "", errors.New("multi factor authentication is not supported")
}
