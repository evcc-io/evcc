package vw

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

// https://identity.vwgroup.io/.well-known/openid-configuration

const (
	// IdentityURI is the VW OIDC identity provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"

	// OauthRevokeURI is used for revoking tokens
	OauthRevokeURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/revoke"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	idtp      *IDTokenProvider
	cvc       *cv.CodeVerifier
	clientID  string
	refresher oauth.TokenRefresher
	ts        *tokenrefreshservice.Service
}

func NewIdentity(log *util.Logger, clientID string, query url.Values, user, password string) *Identity {
	rt := query.Get("response_type")
	fmt.Println(rt)

	var cvc *cv.CodeVerifier
	if strings.Contains(rt, "code") {
		var err error
		cvc, err = cv.CreateCodeVerifier()
		if err != nil {
			panic(err) // should not happen
		}

		query.Set("code_challenge_method", "S256")
		query.Set("code_challenge", cvc.CodeChallengeS256())
	}

	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", IdentityURI, query.Encode())
	fmt.Println(uri)

	return &Identity{
		Helper:    request.NewHelper(log),
		clientID:  clientID,
		idtp:      NewIDTokenProvider(log, uri, user, password),
		refresher: NewTokenRefresher(log, clientID),
		cvc:       cvc,
		ts:        tokenrefreshservice.New(log),
	}
}

func (v *Identity) Login() error {
	token, err := v.login()
	if err != nil {
		return err
	}

	v.TokenSource = oauth.RefreshTokenSource(&token.Token, v)

	return nil
}

// login performs the login using the brand-specific subflow for obtaining the
// id token and finally exchanges the id token for access and refresh tokens
func (v *Identity) login() (Token, error) {
	q, err := v.idtp.Login()

	// at this stage we have the IDK tokens
	idToken := q.Get("id_token")
	code := q.Get("code")
	_ = code

	if err == nil && idToken == "" {
		err = errors.New("missing id_token")
	}

	// if err == nil && code != "" {
	// 	data := url.Values(map[string][]string{
	// 		"code_verifier": {v.cvc.CodeChallengePlain()},
	// 	})

	// 	t, err := v.ts.Exchange(data, idToken, code)
	// 	fmt.Printf("%+v\n", t)

	// 	t, err = v.ts.Refresh(nil, t)
	// 	fmt.Printf("%+v\n", t)

	// 	panic(err)
	// }

	// if v.clientID == "" {
	// 	accessToken := q.Get("access_token")
	// 	if err == nil && accessToken == "" {
	// 		err = errors.New("missing access_token")
	// 	}

	// 	var expires int
	// 	if err == nil {
	// 		expires, err = strconv.Atoi(q.Get("expires_in"))
	// 	}

	// 	return Token{
	// 		Token: oauth2.Token{
	// 			AccessToken: accessToken,
	// 			Expiry:      time.Now().Add(time.Duration(expires) * time.Second),
	// 		},
	// 	}, err
	// }

	var token Token
	if err == nil {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {idToken},
		})

		var req *http.Request
		req, err = request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  v.clientID,
		})

		if err == nil {
			err = v.DoJSON(req, &token)
		}

		// check if token response contained error
		if errT := token.Error(); err != nil && errT != nil {
			err = fmt.Errorf("token exchange: %w", errT)
		}
	}

	return token, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.refresher.RefreshToken(token)
	if err == nil {
		return token, nil
	}

	// re-login
	var res Token
	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return &res.Token, err
}
