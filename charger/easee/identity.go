package easee

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// Token is the Easee Token
type Token struct {
	AccessToken  string    `json:"accessToken"`
	ExpiresIn    float32   `json:"expiresIn"`
	TokenType    string    `json:"tokenType"`
	RefreshToken string    `json:"refreshToken"`
	Valid        time.Time // helper to store validity timestamp
}

// Identity manages the /api/accounts/token and /api/accounts/refresh_token response
type Identity struct {
	*request.Helper
	token Token
}

// registry manages credentials sharing between multiple chargers
var registry = make(map[string]*Identity)

// NewIdentity creates an Easee identity
func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	// get from registry
	if c, ok := registry[user]; ok {
		return c, nil
	}

	c := &Identity{
		Helper: request.NewHelper(log),
	}

	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: user,
		Password: password,
	}

	uri := fmt.Sprintf("%s%s", API, "/accounts/token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		err = c.DoJSON(req, &c.token)
		c.token.Valid = time.Now().Add(time.Second * time.Duration(c.token.ExpiresIn))
	}

	// add to registry
	registry[user] = c

	return c, err
}

// Token returns the if necessary refreshed access token
func (c *Identity) Token() (string, error) {
	var err error
	if c.token.Valid.Before(time.Now()) {
		err = c.refreshToken()
	}

	return c.token.AccessToken, err
}

func (c *Identity) refreshToken() error {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  c.token.AccessToken,
		RefreshToken: c.token.RefreshToken,
	}

	uri := fmt.Sprintf("%s%s", API, "/accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var token Token
	if err == nil {
		err = c.DoJSON(req, &token)
		token.Valid = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
	}

	if err == nil {
		c.token = token
	}

	return err
}
