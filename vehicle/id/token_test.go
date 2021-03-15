package id

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	var tok Token
	str := `{"accesstoken":"access","refreshtoken":"refresh"}`

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
