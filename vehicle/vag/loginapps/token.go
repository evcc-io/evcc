package loginapps

import (
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

// Token is the loginapps token
type Token oauth2.Token

func (t *Token) MarshalJSON() ([]byte, error) {
	tok := struct {
		TokenType    string    `json:"type,omitempty"`
		AccessToken  string    `json:"accesstoken"`
		RefreshToken string    `json:"refreshtoken,omitempty"`
		Expiry       time.Time `json:"expiry,omitempty"`
	}{
		TokenType:    t.TokenType,
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
	}

	return json.Marshal(tok)
}

func (t *Token) UnmarshalJSON(data []byte) error {
	var tok struct {
		TokenType    string    `json:"type,omitempty"`
		AccessToken  string    `json:"accesstoken"`
		RefreshToken string    `json:"refreshtoken,omitempty"`
		Expiry       time.Time `json:"expiry,omitempty"`
	}

	err := json.Unmarshal(data, &tok)
	if err == nil {
		t.AccessToken = tok.AccessToken
		t.RefreshToken = tok.RefreshToken

		t.TokenType = tok.TokenType
		if t.TokenType == "" {
			t.TokenType = "bearer"
		}

		t.Expiry = tok.Expiry
		if t.Expiry.IsZero() {
			t.Expiry = time.Now().Add(time.Hour)
		}
	}

	return err
}
