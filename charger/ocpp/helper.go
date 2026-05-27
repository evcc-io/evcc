package ocpp

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/lorenzodonini/ocpp-go/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
)

// wait waits for a CP roundtrip with timeout
func wait(err error, rc chan error) error {
	if err == nil {
		select {
		case err = <-rc:
			close(rc)
		case <-time.After(Timeout):
			// The ocpp-go dispatcher applies its own per-request timeout, but
			// shares a single timeout context per client: a concurrent
			// CALL_RESULT (e.g. the CP's BootNotification reply) cancels that
			// context, leaving an in-flight request with no timeout at all.
			// Bound the wait here as defense in depth. rc is buffered, so a
			// late response is absorbed and garbage-collected.
			err = api.ErrTimeout
		}

		if oe, ok := errors.AsType[*ocpp.Error](err); ok && oe.Code == ocppj.GenericError {
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
