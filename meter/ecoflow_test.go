package meter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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
		var req struct{ Sn string }
		_ = json.NewDecoder(r.Body).Decode(&req)

		soc := 0
		if req.Sn == leader.Load() {
			soc = 50
		}
		fmt.Fprintf(w, `{"code":"0","data":{"powGetSysGrid":%d,"cmsBattSoc":%d}}`, soc*10, soc)
	}))
	defer srv.Close()

	m, err := NewEcoFlow(t.Context(), "key", "secret", []string{"A", "B", "C"}, "grid", srv.URL, "powGetSysGrid", "", time.Nanosecond, nil, batterySocLimits{}, nil)
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
