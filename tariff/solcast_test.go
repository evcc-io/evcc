package tariff

import (
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
