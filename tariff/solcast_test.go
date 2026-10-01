package tariff

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/require"
)

func TestSolcastActiveWindowOnStartup(t *testing.T) {
	log := util.NewLogger("solcast")
	currentHour := time.Now().Hour()

	// Configure window to be completely outside current hour
	from := (currentHour + 2) % 24
	to := (currentHour + 3) % 24
	if to == 0 {
		to = 24
	}

	sol := &Solcast{
		log:    log,
		site:   "test-site",
		Helper: request.NewHelper(log),
		fromTo: FromTo{From: from, To: to},
		data:   util.NewMonitor[api.Rates](time.Hour),
	}

	done := make(chan error, 1)
	go sol.run(time.Hour, done)

	// Since startup is outside window, it should close done immediately and not error
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for run to complete startup outside window")
	}
}

func TestSolcastPreserveRatesOnError(t *testing.T) {
	log := util.NewLogger("solcast")
	sol := &Solcast{
		log:    log,
		site:   "test-site",
		Helper: request.NewHelper(log),
		fromTo: FromTo{From: 0, To: 24},
		data:   util.NewMonitor[api.Rates](time.Hour),
	}

	// Pre-populate monitor with rates
	initialRates := api.Rates{
		{
			Start: time.Now().Add(time.Hour),
			End:   time.Now().Add(2 * time.Hour),
			Value: 500,
		},
	}
	mergeRatesAfter(sol.data, initialRates, beginningOfDay())

	// Verify rates are readable
	rates, err := sol.Rates()
	require.NoError(t, err)
	require.Len(t, rates, 1)

	// Simulate error after startup: mergeRatesAfter(t.data, nil, beginningOfDay())
	mergeRatesAfter(sol.data, nil, beginningOfDay())

	// Verify rates are STILL readable and not wiped
	ratesAfter, err := sol.Rates()
	require.NoError(t, err)
	require.Len(t, ratesAfter, 1)
	require.Equal(t, float64(500), ratesAfter[0].Value)
}

func TestSolcast429PermanentBodyLog(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response_status": map[string]any{
				"error_code": "TooManyRequests",
				"message":    "You have exceeded your free daily limit.",
			},
		})
	}))
	defer ts.Close()

	log := util.NewLogger("solcast")
	sol := &Solcast{
		log:    log,
		site:   "test-site",
		Helper: request.NewHelper(log),
		fromTo: FromTo{From: 0, To: 24},
		data:   util.NewMonitor[api.Rates](time.Hour),
	}

	// Test the backoff behavior directly by calling GetJSON against our mock server
	var once sync.Once
	done := make(chan error, 1)

	// Simulate 1 run cycle using the mock endpoint url structure
	err := func() error {
		var res struct{}
		err := sol.GetJSON(ts.URL, &res)
		if se, ok := errors.AsType[*request.StatusError](err); ok && se.StatusCode() == http.StatusTooManyRequests {
			var body struct {
				ResponseStatus struct {
					Message string `json:"message"`
				} `json:"response_status"`
			}
			if json.Unmarshal(se.Body(), &body) == nil && body.ResponseStatus.Message != "" {
				// logged
			}
			return backoffPermanentError(err)
		}
		return err
	}()

	require.Error(t, err)
	// reportError should indicate startupFailed on first error
	startupFailed := reportError(&once, done, err)
	require.True(t, startupFailed)
}
