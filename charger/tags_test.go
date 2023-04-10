//go:build !test

package charger

import "log"

// TODO remove when https://github.com/golang/go/issues/52600 becomes available
func init() {
	log.Fatal("running a test without -tags test")
}
