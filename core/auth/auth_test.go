package auth

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
)

type MockSettings struct {
	values map[string]string
}

func (m *MockSettings) String(key string) (string, error) {
	return m.values[key], nil
}

func (m *MockSettings) SetString(key string, value string) {
	m.values[key] = value
}

func TestSetAdminPassword(t *testing.T) {
	mockSettings := &MockSettings{values: make(map[string]string)}
	auth := NewAuth(mockSettings)

	password := "testpassword"

	// first call succeeds
	err := auth.SetAdminPassword(password)
	if err != nil {
		t.Errorf("SetAdminPassword() error = %v", err)
		return
	}

	if mockSettings.values[keys.AdminPassword] == "" {
		t.Errorf("SetAdminPassword() did not store admin password")
	}

	err = auth.SetAdminPassword(password)
	if err == nil {
		t.Errorf("SetAdminPassword() should have failed, admin password already set")
		return
	}
}

func TestRemoveAdminPassword(t *testing.T) {
	mockSettings := &MockSettings{values: make(map[string]string)}
	mockSettings.values[keys.AdminPassword] = "testpassword"
	auth := NewAuth(mockSettings)

	auth.RemoveAdminPassword()

	if mockSettings.values[keys.AdminPassword] != "" {
		t.Errorf("RemoveAdminPassword() did not correctly remove admin password")
	}
}

func TestIsAdminPasswordValid(t *testing.T) {
	mockSettings := &MockSettings{values: make(map[string]string)}
	auth := NewAuth(mockSettings)

	validPw := "testpassword"
	invalidPw := "wrongpassword"

	if auth.IsAdminPasswordValid(validPw) {
		t.Errorf("IsAdminPasswordValid() should have returned false, password not set")
	}

	auth.SetAdminPassword(validPw)

	if !auth.IsAdminPasswordValid(validPw) {
		t.Errorf("IsAdminPasswordValid() should have returned true, matching password")
	}

	if auth.IsAdminPasswordValid(invalidPw) {
		t.Errorf("IsAdminPasswordValid() should have returned false, wrong password")
	}
}

func TestJwtToken(t *testing.T) {
	mockSettings := &MockSettings{values: make(map[string]string)}
	auth := NewAuth(mockSettings)

	lifetime := time.Hour
	tokenString, err := auth.GenerateJwtToken(lifetime)
	if err != nil {
		t.Errorf("GenerateJwtToken() error = %v", err)
		return
	}

	if tokenString == "" {
		t.Errorf("GenerateJwtToken() did not return a token")
	}

	ok, err := auth.ValidateJwtToken(tokenString)
	if !ok || err != nil {
		t.Errorf("ValidateJwtToken() failed to validate token")
	}

	if mockSettings.values[keys.JwtSecret] == "" {
		t.Errorf("GenerateJwtToken() did not store jwtSecret")
	}
}
