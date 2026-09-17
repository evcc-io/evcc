package dataportal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityToken(t *testing.T) {
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "secret-token",
			"expiresIn":   3600,
			"tokenType":   "Bearer",
		})
	}))
	defer srv.Close()

	v := &identity{
		Helper:       request.NewHelper(util.NewLogger("test")),
		clientID:     "id",
		clientSecret: "secret",
		uri:          srv.URL,
	}

	token, err := v.Token()
	require.NoError(t, err)

	// non-standard camelCase fields are mapped into the oauth2 token
	assert.Equal(t, "secret-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)

	// expiry is derived from expiresIn
	assert.WithinDuration(t, time.Now().Add(3600*time.Second), token.Expiry, time.Minute)

	// credentials are sent using the API's camelCase field names
	assert.Equal(t, "id", gotBody["clientId"])
	assert.Equal(t, "secret", gotBody["clientSecret"])
	assert.Contains(t, gotBody["scope"], "pdp-telemetry/battery")
}
