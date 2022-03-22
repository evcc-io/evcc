package id

import (
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

// Token is the VW ID token
type Token oauth2.Token

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		AccessToken  string
		RefreshToken string
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.TokenType = "bearer"
		t.AccessToken = s.AccessToken
		t.RefreshToken = s.RefreshToken
		t.Expiry = time.Now().Add(time.Hour)
	}

	return err
}
