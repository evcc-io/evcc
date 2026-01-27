package ecoflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestPowerStream(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/iot-open/sign/device/quota/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Valid response
		w.Write([]byte(`{
			"code": "0",
			"message": "Success",
			"data": {
				"pv1InputWatts": 100.0,
				"pv2InputWatts": 50.0,
				"invOutputWatts": 300.0,
				"batInputWatts": 120.0,
				"batSoc": 90
			}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	config := map[string]any{
		"uri":       server.URL,
		"sn":        "TEST_SN",
		"accessKey": "key",
		"secretKey": "secret",
	}

	ctx := context.TODO()

	t.Run("PV", func(t *testing.T) {
		conf := cloneMap(config)
		conf["usage"] = "pv"
		m, err := NewPowerStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, 150.0, val) // 100 + 50
	})

	t.Run("Grid", func(t *testing.T) {
		conf := cloneMap(config)
		conf["usage"] = "grid"
		m, err := NewPowerStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, 300.0, val)
	})

	t.Run("Battery", func(t *testing.T) {
		conf := cloneMap(config)
		conf["usage"] = "battery"
		m, err := NewPowerStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		// Test Soc
		b, ok := m.(api.Battery)
		assert.True(t, ok)

		soc, err := b.Soc()
		assert.NoError(t, err)
		assert.Equal(t, 90.0, soc)

		// Test Power (negated)
		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, -120.0, val)
	})
}
