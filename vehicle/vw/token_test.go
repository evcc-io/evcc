package vw

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	var tok Token
	data := `{"access_token":"access","refresh_token":"refresh","token_type":"bearer","expires_in":3600}`

	if err := json.Unmarshal([]byte(data), &tok); err != nil {
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

func TestUnmarshalJSONError(t *testing.T) {
	var tok Token
	data := `{"error":"invalid_request","error_description":"Id token is invalid."}`

	err := json.Unmarshal([]byte(data), &tok)
	if err != nil {
		t.Error(err)
	}

	if err := tok.Error(); err == nil {
		t.Error("missing error")
	}

	if err := tok.Error(); err.Error() != "invalid_request: Id token is invalid." {
		t.Errorf("unexpected error: %s", err.Error())
	}
}
