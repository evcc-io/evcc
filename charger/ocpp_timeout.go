package charger

import (
	"testing"
	"time"
)

var (
	ocppInitialStatusTimeout = 5 * time.Minute
	ocppMessageTimeout       = 30 * time.Second
)

func init() {
	if testing.Testing() {
		ocppInitialStatusTimeout = 1 * time.Second
		ocppMessageTimeout = 1 * time.Second
	}
}
