package server

import (
	"testing"
	"time"
)

func TestTokenCreateValidateRoundtrip(t *testing.T) {
	token, err := CreateToken(time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	if err := ValidateToken(token); err != nil {
		t.Errorf("ValidateToken failed: %v", err)
	}
}