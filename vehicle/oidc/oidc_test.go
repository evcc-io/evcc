package oidc

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	var tok Token
	str := `{"access_token":"access","refresh_token":"refresh","token_type":"bearer","expires_in":3600}`

	if err := json.Unmarshal([]byte(str), &tok); err != nil {
		t.Error(err)
	}

	if tok.AccessToken != "access" {
		t.Error("AccessToken")
	}

	if tok.RefreshToken != "refresh" {
		t.Error("RefreshToken")
	}

	if tok.TokenType != "bearer" {
		t.Error("TokenType")
	}

	if tok.Expiry.IsZero() {
		t.Error("Expiry")
	}
}
