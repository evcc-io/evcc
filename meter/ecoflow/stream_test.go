package ecoflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func cloneMap(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func TestStream(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/iot-open/sign/device/quota/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Valid response
		w.Write([]byte(`{
			"code": "0",
			"message": "Success",
			"data": {
				"powGetPvSum": 150.5,
				"powGetSysGrid": 200.0,
				"powGetBpCms": 50.0,
				"cmsBattSoc": 85.0
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
		m, err := NewStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, 150.5, val)
	})

	t.Run("Grid", func(t *testing.T) {
		conf := cloneMap(config)
		conf["usage"] = "grid"
		m, err := NewStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, 200.0, val)
	})

	t.Run("Battery", func(t *testing.T) {
		conf := cloneMap(config)
		conf["usage"] = "battery"
		m, err := NewStreamFromConfig(ctx, conf)
		assert.NoError(t, err)

		// Test Soc
		b, ok := m.(api.Battery)
		assert.True(t, ok)

		soc, err := b.Soc()
		assert.NoError(t, err)
		assert.Equal(t, 85.0, soc)

		// Test Power (inverted sign)
		val, err := m.CurrentPower()
		assert.NoError(t, err)
		assert.Equal(t, -50.0, val)
	})
}
