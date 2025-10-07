package cardata

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/oauth2"
)

type Token struct {
	*oauth2.Token
	IdToken string `json:"id_token"`
	Gcid    string `json:"gcid"`
}

func (t *Token) TokenEx() *oauth2.Token {
	return t.Token.WithExtra(map[string]any{
		"id_token": t.IdToken,
		"gcid":     t.Gcid,
	})
}

// TokenExtra returns extra string properties of the oauth2.Token
func TokenExtra(t *oauth2.Token, key string) string {
	if v := t.Extra(key); v != nil {
		return v.(string)
	}
	return ""
}

func tokenError(err error) bool {
	return errors.Is(err, api.ErrLoginRequired) || errors.Is(err, api.ErrMissingToken)
}
