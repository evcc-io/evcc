package soc

import "github.com/evcc-io/evcc/core/loadpoint"

// Adapter provides the required methods for interacting with the loadpoint
type Adapter interface {
	loadpoint.API
	Publish(key string, val interface{})
	SocEstimator() *Estimator
}
