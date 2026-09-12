package livewire

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type backend struct {
	logins   atomic.Int32
	sessions atomic.Int32
	token    atomic.Value
	status   func(w http.ResponseWriter, r *http.Request)
}

func newBackend(t *testing.T) (*backend, *Identity) {
	t.Helper()

	b := &backend{}
	b.token.Store("jwt-1")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts.login", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, GigyaAPIKey, r.PostForm.Get("apiKey"))
		assert.Equal(t, "user@example.org", r.PostForm.Get("loginID"))
		assert.Equal(t, "secret", r.PostForm.Get("password"))

		b.logins.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"errorCode": 0, "UID": "uid-1"})
	})
	mux.HandleFunc("POST /api/session", func(w http.ResponseWriter, r *http.Request) {
		var req SessionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "uid-1", req.UID)
		assert.Equal(t, "device-1", req.DeviceUUID)
		assert.Equal(t, DataCenter, req.DataCenter)

		b.sessions.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"jwt": b.token.Load(), "termsAccepted": true})
	})
	mux.HandleFunc("GET /api/getAllbikes/pairingStatus", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+b.token.Load().(string), r.Header.Get("Authorization"))
		assert.Equal(t, Brand, r.URL.Query().Get("brand"))
		assert.Equal(t, "device-1", r.URL.Query().Get("deviceUUID"))
		assert.Equal(t, "Android", r.Header.Get("User-Agent"))

		w.Write([]byte(samplePairStatusAfter))
	})
	mux.HandleFunc("GET /api/bikes/bike-1/charging/status", func(w http.ResponseWriter, r *http.Request) {
		b.status(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	gigya, base, interval := GigyaURL, BaseURL, loginInterval
	GigyaURL, BaseURL, loginInterval = srv.URL, srv.URL+"/api", 0
	t.Cleanup(func() { GigyaURL, BaseURL, loginInterval = gigya, base, interval })

	identity := NewIdentity(util.NewLogger("test"), "user@example.org", "secret", "device-1")

	return b, identity
}

func TestLoginAndVehicles(t *testing.T) {
	b, identity := newBackend(t)
	require.NoError(t, identity.Login())

	bikes, err := NewAPI(util.NewLogger("test"), identity).Vehicles()
	require.NoError(t, err)
	require.Len(t, bikes, 1)
	assert.Equal(t, "100000001", bikes[0].ID)
	assert.True(t, bikes[0].PairingStatus)
	assert.Equal(t, int32(1), b.logins.Load())
	assert.Equal(t, int32(1), b.sessions.Load())
}

func TestStatus(t *testing.T) {
	b, identity := newBackend(t)
	b.status = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleStatusIdle))
	}

	res, err := NewAPI(util.NewLogger("test"), identity).Status("bike-1")
	require.NoError(t, err)
	assert.False(t, res.ChargingStatus)
	assert.Equal(t, 65.0, res.BatteryPercentage)
	assert.Equal(t, int64(80), res.MaxLimit)
}

func TestProviderConvertsMiles(t *testing.T) {
	b, identity := newBackend(t)
	b.status = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleStatusIdle))
	}

	p := NewProvider(NewAPI(util.NewLogger("test"), identity), "bike-1", time.Minute)

	rng, err := p.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(108), rng)

	odo, err := p.Odometer()
	require.NoError(t, err)
	assert.InDelta(t, 886.2, odo, 0.1)

	soc, err := p.Soc()
	require.NoError(t, err)
	assert.Equal(t, 65.0, soc)

	status, err := p.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)

	limit, err := p.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(80), limit)
}

func TestProviderCharging(t *testing.T) {
	b, identity := newBackend(t)
	b.status = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleStatusCharging))
	}

	p := NewProvider(NewAPI(util.NewLogger("test"), identity), "bike-1", time.Minute)

	status, err := p.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	// timeToMaxLimit is minutes: 45 for 65 -> 80 % on the onboard charger
	finish, err := p.FinishTime()
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(45*time.Minute), finish, 5*time.Second)
}

func TestErrorEnvelopeIsAsleep(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusBadRequest} {
		b, identity := newBackend(t)
		b.status = func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": "3000", "description": "Command manager api failed"},
			})
		}

		_, err := NewAPI(util.NewLogger("test"), identity).Status("bike-1")
		assert.ErrorIs(t, err, api.ErrAsleep, "status %d", code)
		assert.ErrorIs(t, err, api.ErrTimeout, "status %d", code)
	}
}

func TestUnauthorizedTriggersRelogin(t *testing.T) {
	b, identity := newBackend(t)
	require.NoError(t, identity.Login())

	b.status = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer jwt-2" {
			b.token.Store("jwt-2")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"bikeChargingData": map[string]any{"batteryPercentage": 50}})
	}

	res, err := NewAPI(util.NewLogger("test"), identity).Status("bike-1")
	require.NoError(t, err)
	assert.Equal(t, 50.0, res.BatteryPercentage)
	assert.Equal(t, int32(2), b.logins.Load())
}

func TestLoginThrottled(t *testing.T) {
	b, identity := newBackend(t)
	loginInterval = time.Second

	require.NoError(t, identity.Login())
	identity.invalidate("jwt-1")

	_, err := identity.Token()
	assert.ErrorContains(t, err, "throttled")
	assert.Equal(t, int32(1), b.logins.Load())
}
