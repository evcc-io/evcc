package vag

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// Token is an OAuth2-compatible token that supports the expires_in attribute
type Token struct {
	oauth2.Token
	IDToken string `json:"id_token,omitempty"`
	err     error
}

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		oauth2.Token
		IDToken          string `json:"id_token,omitempty"`
		ExpiresIn        int64  `json:"expires_in,omitempty"`
		Error            *string
		ErrorDescription *string `json:"error_description"`
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.Token = s.Token
		t.IDToken = s.IDToken

		if s.Expiry.IsZero() && s.ExpiresIn != 0 {
			t.Expiry = time.Now().Add(time.Second * time.Duration(s.ExpiresIn))
		}

		if s.Error != nil && s.ErrorDescription != nil {
			t.err = fmt.Errorf("%s: %s", *s.Error, *s.ErrorDescription)
		}
	}

	return err
}

func (t *Token) Error() error {
	return t.err
}
