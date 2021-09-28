package oauth

import (
	"encoding/json"

	"github.com/evcc-io/evcc/util/oauth/internal"
	"golang.org/x/oauth2"
)

// Token is an OAuth token that supports the expires_in attribute
type Token oauth2.Token

func (t *Token) UnmarshalJSON(data []byte) error {
	var o internal.Token

	err := json.Unmarshal(data, &o)
	if err == nil {
		*t = (Token)(o.Token)
	}

	return err
}
