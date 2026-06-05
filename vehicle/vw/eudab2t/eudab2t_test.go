package eudab2t

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "test-marketplace-key"

// testAPI returns a client pointed at the given test server
func testAPI(url string) *API {
	v := NewAPI(util.NewLogger("test"), testKey)
	v.baseURL = url
	return v
}

func TestData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subscription/vehicles/WAU3FBFR9BE029956/data", r.URL.Path)
		assert.Equal(t, testKey, r.Header.Get("x-marketplace-api-key"))

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":{"state_of_charge":"82","cruising_range_primary_engine":"310","plug_state":"connected","charging_state":"charging"}}`)
	}))
	defer srv.Close()

	data, err := testAPI(srv.URL).Data("WAU3FBFR9BE029956")
	require.NoError(t, err)
	assert.Equal(t, "82", data[FieldSoc])
	assert.Equal(t, "310", data[FieldRangePrimary])
}

func TestSubscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/subscription/vehicles", r.URL.Path)
		assert.Equal(t, testKey, r.Header.Get("x-marketplace-api-key"))

		var body VINList
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []string{"VIN1"}, body.Add)
		assert.Equal(t, []string{"VIN2"}, body.Remove)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	require.NoError(t, testAPI(srv.URL).Subscribe([]string{"VIN1"}, []string{"VIN2"}))
}

func TestStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subscription/vehicles/status", r.URL.Path)
		assert.Equal(t, testKey, r.Header.Get("x-marketplace-api-key"))
		assert.Equal(t, "idk-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"vehicle":"WAU3FBFR9BE029956","status":"granted"}]`)
	}))
	defer srv.Close()

	res, err := testAPI(srv.URL).Status("idk-token")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "WAU3FBFR9BE029956", res[0].Vehicle)
	assert.Equal(t, "granted", res[0].Status)
}

func TestProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":{"state_of_charge":"82","cruising_range_primary_engine":"310","mileage":"12345","settings.target_soc":"90","plug_state":"connected","charging_state":"charging"}}`)
	}))
	defer srv.Close()

	p := &Provider{dataG: util.Cached(func() (map[string]string, error) {
		return testAPI(srv.URL).Data("VIN1")
	}, time.Minute)}

	soc, err := p.Soc()
	require.NoError(t, err)
	assert.Equal(t, 82.0, soc)

	rng, err := p.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(310), rng)

	odo, err := p.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 12345.0, odo)

	limit, err := p.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(90), limit)

	status, err := p.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
}
