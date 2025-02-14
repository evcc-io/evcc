package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestForecastSlots(t *testing.T) {
	tf := Tariff{
		embed: new(embed),
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	clock := clock.NewMock()
	fc := api.Rates{
		{
			Start: clock.Now().Add(1 * time.Hour),
			End:   clock.Now().Add(2 * time.Hour),
			Price: 10,
		},
		{
			Start: clock.Now().Add(4 * time.Hour),
			End:   clock.Now().Add(5 * time.Hour),
			Price: 40,
		},
	}

	done := make(chan error, 1)
	go tf.run(func() (string, error) {
		j, err := fc.MarshalMQTT()
		return string(j), err
	}, done, time.Hour)

	require.NoError(t, <-done)

	rr, err := tf.forecastRates()
	require.NoError(t, err)
	require.Equal(t, 4, len(rr))
}
