package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestAverage(t *testing.T) {
	clock := clock.NewMock()
	clock.Add(30 * time.Minute)

	var rr api.Rates
	for i := range 5 {
		rr = append(rr, api.Rate{
			Start: clock.Now(),
			End:   clock.Now().Add(SlotDuration),
			Value: float64(i + 1),
		})
		clock.Add(SlotDuration)
	}

	res := averageSlots(rr, time.Hour)

	rr[0].Value = 1.5
	rr[1].Value = 1.5
	rr[2].Value = 4.0
	rr[3].Value = 4.0
	rr[4].Value = 4.0

	require.Equal(t, rr, res)
}
