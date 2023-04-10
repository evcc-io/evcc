//go:build !test

package charger

import "time"

const (
	ocppConnectTimeout = 5 * time.Minute
	ocppTimeout        = 2 * time.Minute
)
