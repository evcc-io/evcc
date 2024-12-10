package polestar

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const (
	OAuthURI    = "https://polestarid.eu.polestar.com"
	ClientID    = "l3oopkc_10"
	RedirectURI = "https://www.polestar.com/sign-in-callback"
)

type Identity struct {
	*request.Helper
	user, password string
	jar            *cookiejar.Jar
	log            *util.Logger
}

// NewIdentity creates Polestar identity
func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
		log:      log,
	}

	log.DEBUG.Printf("initializing polestar identity with user: %s", user)

	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}
	v.jar = jar
	v.Client.Jar = jar

	token, err := v.login()
	if err != nil {
		return nil, err
	}

	v.Client.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(token),
		Base:   v.Client.Transport,
	}

	return v, nil
}

// generates code verifier for PKCE
func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

// generates code challenge from verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return strings.TrimRight(base64.URLEncoding.EncodeToString(hash[:]), "=")
}

func (v *Identity) login() (*oauth2.Token, error) {
	state := lo.RandomString(16, lo.AlphanumericCharset)
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Build authorization URI with all required scopes
	authURL := fmt.Sprintf("%s/as/authorization.oauth2"+
		"?client_id=%s"+
		"&redirect_uri=%s"+
		"&response_type=code"+
		"&state=%s"+
		"&scope=openid%%20profile%%20email"+
		"&code_challenge=%s"+
		"&code_challenge_method=S256",
		OAuthURI, ClientID, RedirectURI, state, codeChallenge)

	// Get resume path with browser-like headers
	req, err := request.New(http.MethodGet, authURL, nil, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, err
	}

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	v.log.TRACE.Printf("auth response URL: %s", resp.Request.URL.String())
	resp.Body.Close()

	// Extract resume path from redirect URL
	if resp.Request.URL == nil {
		return nil, fmt.Errorf("no redirect url")
	}

	// First we get redirected to the login page
	if strings.Contains(resp.Request.URL.Path, "/PolestarLogin/login") {
		// Extract resumePath from the login URL
		resumePath := resp.Request.URL.Query().Get("resumePath")
		if resumePath == "" {
			return nil, fmt.Errorf("resume path not found in login URL: %s", resp.Request.URL.String())
		}
		v.log.TRACE.Printf("got resume path: %s", resumePath)

		// Submit credentials directly to the login endpoint
		loginURL := fmt.Sprintf("%s/as/%s/resume/as/authorization.ping", OAuthURI, resumePath)
		data := url.Values{
			"pf.username": []string{v.user},
			"pf.pass":     []string{v.password},
			"client_id":   []string{ClientID},
		}

		req, err = request.New(http.MethodPost, loginURL, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "application/json",
		})
		if err != nil {
			return nil, err
		}

		resp, err = v.Do(req)
		if err != nil {
			return nil, err
		}
		v.log.TRACE.Printf("login response URL: %s", resp.Request.URL.String())
		resp.Body.Close()

		if resp.Request.URL == nil {
			return nil, fmt.Errorf("no redirect url after login")
		}
	}

	// After login, we should get the authorization code directly
	query := resp.Request.URL.Query()
	code := query.Get("code")
	if code != "" {
		v.log.TRACE.Printf("got authorization code directly")
		goto exchange
	}

	return nil, fmt.Errorf("authorization code not found in URL: %s", resp.Request.URL.String())

exchange:
	// Exchange code for token
	data := url.Values{
		"grant_type":    []string{"authorization_code"},
		"code":          []string{code},
		"code_verifier": []string{codeVerifier},
		"client_id":     []string{ClientID},
		"redirect_uri":  []string{RedirectURI},
	}

	var token Token
	req, err = request.New(http.MethodPost, OAuthURI+"/as/token.oauth2",
		strings.NewReader(data.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "application/json",
		},
	)
	if err == nil {
		err = v.DoJSON(req, &token)
	}

	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    "Bearer",
		RefreshToken: token.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
	}, err
}

// TokenSource implements oauth.TokenSource
func (v *Identity) TokenSource() oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, v)
}

// Token implements oauth.TokenSource
func (v *Identity) Token() (*oauth2.Token, error) {
	return v.login()
}
