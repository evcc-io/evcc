package chargepoint

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// TestDoJSON_ReauthOn401 verifies that doJSON transparently re-authenticates
// and retries when the server returns HTTP 401.
func TestDoJSON_ReauthOn401(t *testing.T) {
	var accountCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/driver/profile/account/login":
			// Login endpoint — return fresh tokens.
			json.NewEncoder(w).Encode(map[string]any{
				"sessionId":    "new-session",
				"ssoSessionId": "new-sso-session",
				"user":         map[string]any{"userId": 42},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/v1/driver/profile/user":
			// Account endpoint — 401 on first call, 200 on retry.
			if accountCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]any{"error": "unauthorized"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"user": map[string]any{"userId": 42},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	base := srv.URL + "/"

	identity := &Identity{
		Helper:      request.NewHelper(util.NewLogger("test")),
		settingsKey: "chargepoint.test",
		deviceData:  newDeviceData("test@example.com"),
		cfg: &globalConfig{
			Region: "US",
			EndPoints: configEndpoints{
				Accounts:    endpointValue{Value: base},
				Chargers:    endpointValue{Value: base},
				InternalAPI: endpointValue{Value: base},
				WebServices: endpointValue{Value: base},
			},
		},
		identityState: identityState{
			Username:     "test@example.com",
			Password:     "password",
			SessionID:    "old-session",
			SSOSessionID: "old-sso-session",
			Region:       "US",
		},
	}

	api := NewAPI(util.NewLogger("test"), identity)

	userID, err := api.Account()
	if err != nil {
		t.Fatalf("Account() error: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID: got %d, want 42", userID)
	}
	if n := accountCalls.Load(); n != 2 {
		t.Errorf("account endpoint calls: got %d, want 2", n)
	}
	if identity.SessionID != "new-session" {
		t.Errorf("SessionID after reauth: got %q, want %q", identity.SessionID, "new-session")
	}
}
