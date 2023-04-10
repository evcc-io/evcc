//go:build test

package charger

import "time"

const (
	ocppConnectTimeout = 1 * time.Second
	ocppTimeout        = 1 * time.Second
)
