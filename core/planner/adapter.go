package planner

import (
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/soc"
)

// Adapter provides the required methods for interacting with the loadpoint
type Adapter interface {
	loadpoint.API
	Publish(key string, val interface{})
	SocEstimator() *soc.Estimator
}
