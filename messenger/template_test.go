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
	t.Run("positive_delay", func(t *testing.T) {
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
			require.GreaterOrEqual(t, received.Sub(start), 500*time.Millisecond)
		case <-time.After(3 * time.Second):
			t.Fatal("ntfy request not received")
		}
	})

	t.Run("negative_delay_clamped_to_zero", func(t *testing.T) {
		ntfy := newTestNtfyWithDelay(t, -5)
		require.Equal(t, time.Duration(0), ntfy.delay)
	})

	t.Run("zero_delay_stays_zero", func(t *testing.T) {
		ntfy := newTestNtfyWithDelay(t, 0)
		require.Equal(t, time.Duration(0), ntfy.delay)
	})

	t.Run("large_delay_is_clamped", func(t *testing.T) {
		ntfy := newTestNtfyWithDelay(t, 10_000)
		require.Equal(t, maxDelay, ntfy.delay)
	})
}

func newTestNtfyWithDelay(t *testing.T, delay int) *Ntfy {
	t.Helper()

	messenger, err := NewNtfyFromConfig(map[string]any{
		"uri":      "http://example.com",
		"delay":    delay,
		"tags":     "",
		"priority": "default",
	})
	require.NoError(t, err)

	ntfy, ok := messenger.(*Ntfy)
	require.True(t, ok, "expected *Ntfy messenger type")

	return ntfy
}
