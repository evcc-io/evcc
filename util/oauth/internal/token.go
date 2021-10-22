package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// Token is an OAuth2 token which support decoding the expires_in attribute and return content errors
type Token struct {
	oauth2.Token
	ExpiresIn int `json:"expires_in"` // expiration time in seconds
}

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		oauth2.Token
		ExpiresIn int64 `json:"expires_in,omitempty"`
		// used by VW
		Error            *string
		ErrorDescription *string `json:"error_description"`
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.Token = s.Token

		if s.Expiry.IsZero() && s.ExpiresIn != 0 {
			t.Expiry = time.Now().Add(time.Second * time.Duration(s.ExpiresIn))
		}

		if s.Error != nil && s.ErrorDescription != nil {
			err = fmt.Errorf("%s: %s", *s.Error, *s.ErrorDescription)
		}
	}

	return err
}
