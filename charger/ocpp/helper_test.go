package ocpp

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/assert"
)

func TestWaitMapsNoClientToTimeout(t *testing.T) {
	// dispatcher returns this verbatim when the CP disconnected before the request could be queued
	err := errors.New("cannot send request 4711, no client ABC123 exists")
	got := wait(err, nil)
	assert.ErrorIs(t, got, api.ErrTimeout)
}

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
