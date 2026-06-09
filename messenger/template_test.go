package messenger

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
)

var acceptable = []string{
	// api.ErrMissingCredentials.Error(),
	// api.ErrMissingToken.Error(),
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Messenger, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig(t.Context(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}

func TestNtfyDelay(t *testing.T) {
	requestTime := make(chan time.Time, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTime <- time.Now()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	messenger, err := NewNtfyFromConfig(map[string]any{
		"uri":      server.URL,
		"delay":    1,
		"tags":     "",
		"priority": "default",
	})
	require.NoError(t, err)

	start := time.Now()
	messenger.Send("title", "message")

	select {
	case received := <-requestTime:
		require.GreaterOrEqual(t, received.Sub(start), time.Second)
	case <-time.After(2 * time.Second):
		t.Fatal("ntfy request not received")
	}
}

func TestNtfyDelayNonBlocking(t *testing.T) {
	requestTime := make(chan time.Time, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTime <- time.Now()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	messenger, err := NewNtfyFromConfig(map[string]any{
		"uri":      server.URL,
		"delay":    1,
		"tags":     "",
		"priority": "default",
	})
	require.NoError(t, err)

	start := time.Now()
	messenger.Send("title", "message")
	elapsed := time.Since(start)

	// Send must be non-blocking relative to the configured delay.
	require.Less(t, elapsed, 100*time.Millisecond)

	// Still ensure the HTTP request arrives after the configured delay.
	select {
	case received := <-requestTime:
		require.GreaterOrEqual(t, received.Sub(start), time.Second)
	case <-time.After(2 * time.Second):
		t.Fatal("ntfy request not received")
	}
}
