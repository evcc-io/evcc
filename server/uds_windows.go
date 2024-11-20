//go:build windows

package server

import "github.com/evcc-io/evcc/core/site"

// HealthListener attaches listener to unix domain socket
func HealthListener(_ site.API) {
	// nop
}
