//go:build darwin && !cgo

package service

import "github.com/evcc-io/evcc/server/service"

// serialPortOptions falls back to plain port names: the darwin enumerator needs IOKit via cgo
func serialPortOptions() []service.Option {
	return plainSerialPorts()
}
