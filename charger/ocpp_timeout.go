package charger

import (
	"testing"
	"time"
)

var (
	ocppConnectTimeout = 5 * time.Minute
	ocppTimeout        = 2 * time.Minute
)

func init() {
	if testing.Testing() {
		ocppConnectTimeout = 1 * time.Second
		ocppTimeout = 1 * time.Second
	}
}
