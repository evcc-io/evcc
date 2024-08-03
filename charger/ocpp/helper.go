package ocpp

import (
	"time"

	"github.com/evcc-io/evcc/api"
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
