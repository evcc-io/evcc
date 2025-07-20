package ocpp

import (
	"testing"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/assert"
)

func TestSortByAge(t *testing.T) {
	assert.Equal(t, []types.MeterValue{
		{Timestamp: nil},
		{Timestamp: types.NewDateTime(time.UnixMilli(1))},
		{Timestamp: types.NewDateTime(time.UnixMilli(2))},
		{Timestamp: types.NewDateTime(time.UnixMilli(3))},
	}, sortByAge([]types.MeterValue{
		{Timestamp: types.NewDateTime(time.UnixMilli(3))},
		{Timestamp: types.NewDateTime(time.UnixMilli(1))},
		{Timestamp: nil},
		{Timestamp: types.NewDateTime(time.UnixMilli(2))},
	}))
}
