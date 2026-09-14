package homeassistant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupervisorToken(t *testing.T) {
	t.Setenv(SupervisorToken, "test_supervisor_token")

	ts, ok := supervisorTokenSource(SupervisorURI)
	require.True(t, ok)

	tok, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "test_supervisor_token", tok.AccessToken)

	// SupervisorURI variant with trailing slash should still match
	tsSlash, ok := supervisorTokenSource(SupervisorURI + "/")
	require.True(t, ok)

	tokSlash, err := tsSlash.Token()
	require.NoError(t, err)
	assert.Equal(t, "test_supervisor_token", tokSlash.AccessToken)

	// empty uri does not match supervisor
	_, ok = supervisorTokenSource("")
	assert.False(t, ok)

	// other uri does not match supervisor
	_, ok = supervisorTokenSource("http://homeassistant.local:8123")
	assert.False(t, ok)

	// from config with SupervisorURI
	ts3, err := NewHomeAssistantFromConfig(map[string]any{"uri": SupervisorURI})
	require.NoError(t, err)

	tok3, err := ts3.Token()
	require.NoError(t, err)
	assert.Equal(t, "test_supervisor_token", tok3.AccessToken)

	// when SUPERVISOR_TOKEN is unset, NewHomeAssistantFromConfig should fall back to the standard OAuth token source
	t.Setenv(SupervisorToken, "")

	ts4, err := NewHomeAssistantFromConfig(map[string]any{"uri": SupervisorURI})
	require.NoError(t, err)

	_, err = ts4.Token()
	require.Error(t, err)
	var elr *api.ErrLoginRequired
	assert.ErrorAs(t, err, &elr)
	// connection requires uri
	_, err = NewConnection(util.NewLogger("test"), "", "", false)
	assert.Error(t, err)

	conn, err := NewConnection(util.NewLogger("test"), SupervisorURI, "", false)
	require.NoError(t, err)
	assert.Equal(t, SupervisorURI, conn.URI())

	// test authenticated request using connection
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"entity_id":"sensor.test","state":"10"}]`))
	}))
	defer srv.Close()

	testConn, err := NewConnection(util.NewLogger("test"), srv.URL, "", false)
	require.NoError(t, err)
	// override instance token source directly or test via proxyInstance
	testConn.instance.TokenSource = ts

	states, err := testConn.GetStates()
	require.NoError(t, err)
	assert.Len(t, states, 1)
	assert.Equal(t, "Bearer test_supervisor_token", authHeader)
}

func TestSupervisorDiscovery(t *testing.T) {
	t.Setenv(SupervisorToken, "test_supervisor_token")

	addInstance(SupervisorInstance, SupervisorURI)
	assert.Equal(t, SupervisorURI, instanceUriByName(SupervisorInstance))
	assert.Equal(t, SupervisorInstance, instanceNameByUri(SupervisorURI))
}
