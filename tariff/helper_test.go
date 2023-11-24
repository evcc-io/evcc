package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
)

func TestLevel(t *testing.T) {
	var r api.Rates

	for i := 0; i < 24; i++ {
		r = append(r, api.Rate{
			Start: now.BeginningOfHour().Add(time.Duration(i) * time.Hour),
			End:   now.BeginningOfHour().Add(time.Duration(i+1) * time.Hour),
			Price: float64(i),
		})
	}

	assert.Len(t, LevelRates(r, 0.6), 24)
}
