package oauth

import (
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

// Token is an OAuth token that supports the expires_in attribute
type Token oauth2.Token

func (t *Token) UnmarshalJSON(data []byte) error {
	var o struct {
		oauth2.Token
		ExpiresIn int64 `json:"expires_in,omitempty"`
	}

	err := json.Unmarshal(data, &o)
	if err == nil {
		*t = (Token)(o.Token)

		if o.Expiry.IsZero() && o.ExpiresIn != 0 {
			t.Expiry = time.Now().Add(time.Second * time.Duration(o.ExpiresIn))
		}
	}

	return err
}
