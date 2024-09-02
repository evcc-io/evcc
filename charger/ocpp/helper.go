package ocpp

import (
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// Wait waits for a CP roundtrip with timeout
func Wait(err error, rc chan error, timeout time.Duration) error {
	if err == nil {
		select {
		case err = <-rc:
			close(rc)
		case <-time.After(timeout):
			err = api.ErrTimeout
		}
	}
	return err
}

func sortByAge(values []types.MeterValue) []types.MeterValue {
	return slices.SortedFunc(slices.Values(values), func(a, b types.MeterValue) int {
		var at, bt time.Time
		if a.Timestamp != nil {
			at = a.Timestamp.Time
		}
		if b.Timestamp != nil {
			bt = b.Timestamp.Time
		}
		return at.Compare(bt)
	})
}

// hasProperty checks if comma-separated string contains given string ignoring white spaces
func hasProperty(props, prop string) bool {
	return slices.ContainsFunc(strings.Split(props, ","), func(s string) bool {
		return strings.HasPrefix(strings.ReplaceAll(s, " ", ""), prop)
	})
}
