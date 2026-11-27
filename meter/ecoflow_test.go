package meter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEcoflowReserveLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit float64
		lo    float64
		want  float64
	}{
		{name: "above discharge limit", limit: 50, lo: 7, want: 50},
		{name: "at required distance", limit: 10, lo: 7, want: 10},
		{name: "below required distance", limit: 5, lo: 7, want: 10},
		{name: "discharge limit raised", limit: 10, lo: 15, want: 18},
		{name: "charge mode", limit: 100, lo: 7, want: 100},
		{name: "distance exceeds full reserve", limit: 10, lo: 99, want: 100},
		{name: "zero discharge limit", limit: 0, lo: 0, want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ecoflowReserveLimit(tc.limit, tc.lo))
		})
	}
}

func TestEcoflowLeaderChange(t *testing.T) {
	var leader atomic.Value
	leader.Store("B")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/iot-open/sign/device/quota", r.URL.Path, "serial lists must not look up the main device")

		var req struct{ Sn string }
		_ = json.NewDecoder(r.Body).Decode(&req)

		soc := 0
		if req.Sn == leader.Load() {
			soc = 50
		}
		fmt.Fprintf(w, `{"code":"0","data":{"powGetSysGrid":%d,"cmsBattSoc":%d}}`, soc*10, soc)
	}))
	defer srv.Close()

	m, err := NewEcoFlow("key", "secret", []string{"A", "B", "C"}, "grid", srv.URL, "powGetSysGrid", "", true, time.Nanosecond, nil, batterySocLimits{}, nil)
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 500.0, p)
	assert.Equal(t, "B", m.serial())

	leader.Store("C")
	time.Sleep(time.Millisecond)

	p, err = m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 500.0, p)
	assert.Equal(t, "C", m.serial())
}

func TestEcoflowMainDevice(t *testing.T) {
	var main atomic.Value
	main.Store("B")

	var lookups atomic.Int32
	var failing atomic.Bool
	var read, written, reserve atomic.Value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/iot-open/sign/device/system/main/sn" {
			lookups.Add(1)
			assert.Equal(t, "A", r.URL.Query().Get("sn"))
			if failing.Load() {
				fmt.Fprint(w, `{"code":"1006","message":"device not found"}`)
				return
			}
			fmt.Fprintf(w, `{"code":"0","data":{"sn":%q}}`, main.Load())
			return
		}

		var req struct {
			Sn     string
			Params map[string]any
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		if r.Method == http.MethodPut {
			written.Store(req.Sn)
			reserve.Store(req.Params["cfgBackupReverseSoc"])
			fmt.Fprint(w, `{"code":"0","message":"Success"}`)
			return
		}
		read.Store(req.Sn)

		// followers report zero system values
		var pwr, soc int
		if req.Sn == main.Load() {
			pwr, soc = 500, 50
		}
		fmt.Fprintf(w, `{"code":"0","data":{"sysEnergyStreamReport-gridPwr":%d,"powGetBpCms":%d,"cmsBattSoc":%d,"cmsMinDsgSoc":10}}`, pwr, pwr, soc)
	}))
	defer srv.Close()

	t.Run("disabled", func(t *testing.T) {
		lookups.Store(0)

		m, err := NewEcoFlow("key", "secret", []string{"A"}, "grid", srv.URL, "sysEnergyStreamReport-gridPwr", "", false, time.Nanosecond, nil, batterySocLimits{}, nil)
		require.NoError(t, err)

		p, err := m.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 0.0, p)
		assert.Equal(t, "A", m.serial())
		assert.Zero(t, lookups.Load())
	})

	t.Run("main device", func(t *testing.T) {
		lookups.Store(0)

		m, err := NewEcoFlow("key", "secret", []string{"A"}, "battery", srv.URL, "powGetBpCms", "cmsBattSoc", true, time.Minute, nil, batterySocLimits{MinSoc: 20}, nil)
		require.NoError(t, err)

		// reads stay on the configured serial
		p, err := m.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 0.0, p)
		assert.Equal(t, "A", read.Load())

		// commands follow the main device, hold uses its soc
		ctrl, ok := api.Cap[api.BatteryController](m)
		require.True(t, ok)
		require.NoError(t, ctrl.SetBatteryMode(api.BatteryHoldDischarge))
		assert.Equal(t, "B", read.Load(), "discharge limit")
		assert.Equal(t, "B", written.Load(), "backup reserve")
		assert.EqualValues(t, 50, reserve.Load(), "hold soc")

		// one lookup per command
		assert.EqualValues(t, 1, lookups.Load())
	})

	t.Run("lookup fails", func(t *testing.T) {
		lookups.Store(0)
		failing.Store(true)
		defer failing.Store(false)

		m, err := NewEcoFlow("key", "secret", []string{"A"}, "grid", srv.URL, "sysEnergyStreamReport-gridPwr", "", true, time.Nanosecond, nil, batterySocLimits{}, nil)
		require.NoError(t, err)

		// falls back to the configured serial
		assert.Equal(t, "A", m.serial())
		assert.EqualValues(t, 1, lookups.Load())
	})

	t.Run("lookup fails after success", func(t *testing.T) {
		lookups.Store(0)
		main.Store("B")

		m, err := NewEcoFlow("key", "secret", []string{"A"}, "grid", srv.URL, "sysEnergyStreamReport-gridPwr", "", true, time.Nanosecond, nil, batterySocLimits{}, nil)
		require.NoError(t, err)
		assert.Equal(t, "B", m.serial())

		// keeps the last main device
		failing.Store(true)
		defer failing.Store(false)
		time.Sleep(time.Millisecond)

		assert.Equal(t, "B", m.serial())
		assert.EqualValues(t, 2, lookups.Load())
	})
}
