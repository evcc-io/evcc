package ocpp

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// bind the central system to an ephemeral port so this test binary does not
	// contend with the charger package test binary for the fixed default port
	// when both run in parallel under `go test ./...`
	Init(Config{Port: 0}, "")
	os.Exit(m.Run())
}

func TestExternalUrl(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"", ""},
		{"http://example.com", "ws://example.com:8887"},
		{"https://example.com:443", "ws://example.com:8887"},
		{"http://10.20.30.40:7070/path", "ws://10.20.30.40:8887/path"},
	}

	for _, tt := range tests {
		externalUrl = tt.input
		if result := ExternalUrl(); result != tt.expected {
			t.Errorf("ExternalUrl(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestConfigureDateTimeGranularity(t *testing.T) {
	origGranularity := dateTimeGranularity
	origTypeFormat := types.DateTimeFormat
	t.Cleanup(func() {
		dateTimeGranularity = origGranularity
		types.DateTimeFormat = origTypeFormat
	})

	ts := types.NewDateTime(time.Date(2026, 6, 23, 12, 45, 9, 404_000_000, time.UTC))

	configureDateTimeGranularity(DateTimeGranularitySeconds)
	b, err := json.Marshal(ts)
	require.NoError(t, err)
	require.JSONEq(t, `"2026-06-23T12:45:09Z"`, string(b))
	require.Equal(t, DateTimeGranularitySeconds, CurrentConfig().DateTimeGranularity)

	configureDateTimeGranularity(DateTimeGranularityMilliseconds)
	b, err = json.Marshal(ts)
	require.NoError(t, err)
	require.JSONEq(t, `"2026-06-23T12:45:09.404Z"`, string(b))
	require.Equal(t, DateTimeGranularityMilliseconds, CurrentConfig().DateTimeGranularity)
}
