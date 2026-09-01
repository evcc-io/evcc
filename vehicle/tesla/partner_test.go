package tesla

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestRegisterPartner(t *testing.T) {
	var tokenForm url.Values
	var registered []string
	status := http.StatusOK
	errBody := `{"error":"key not found"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/token":
			require.NoError(t, r.ParseForm())
			tokenForm = r.PostForm
			_, _ = w.Write([]byte(`{"access_token":"partner","token_type":"Bearer","expires_in":3600}`))

		case "/api/1/partner_accounts":
			require.Equal(t, "Bearer partner", r.Header.Get("Authorization"))

			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			registered = append(registered, body["domain"])

			w.WriteHeader(status)
			if status != http.StatusOK {
				_, _ = w.Write([]byte(errBody))
				return
			}
			_, _ = w.Write([]byte(`{"response":{}}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	audience := srv.URL + "/"
	tokenURL, fleetAudiences = srv.URL+"/token", []string{audience}

	log := util.NewLogger("test")

	require.NoError(t, registerPartner(t.Context(), log, "id", "secret", "evcc.example"))
	require.Equal(t, []string{"evcc.example"}, registered)
	require.Equal(t, "client_credentials", tokenForm.Get("grant_type"))
	require.Equal(t, "id", tokenForm.Get("client_id"))
	require.Equal(t, "secret", tokenForm.Get("client_secret"))
	require.Equal(t, srv.URL, tokenForm.Get("audience"), "audience without trailing slash")

	// one region is enough
	fleetAudiences = []string{srv.URL + "/missing/", audience}
	require.NoError(t, registerPartner(t.Context(), log, "id", "secret", "evcc.example"))

	// tesla rejects the domain, its explanation is passed on
	fleetAudiences = []string{audience}
	status = http.StatusUnprocessableEntity
	err := registerPartner(t.Context(), log, "id", "secret", "evcc.example")
	require.ErrorContains(t, err, "422")
	require.ErrorContains(t, err, "key not found")

	// key bound to another app or domain
	errBody = `{"error":"Validation failed: Public key hash has already been taken"}`
	err = registerPartner(t.Context(), log, "id", "secret", "evcc.example")
	require.ErrorIs(t, err, errKeyTaken)

	// domain bound to another app
	errBody = `{"error":"Validation failed: Domain has already been taken"}`
	err = registerPartner(t.Context(), log, "id", "secret", "evcc.example")
	require.ErrorContains(t, err, "evcc.example is registered with another Tesla app")
}
