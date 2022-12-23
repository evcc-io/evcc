package core

import (
	"github.com/evcc-io/evcc/core/soc"
)

var _ soc.Adapter = (*adapter)(nil)

type adapter struct {
	*Loadpoint
}

func (a *adapter) Publish(key string, val interface{}) {
	a.Loadpoint.publish(key, val)
}

func (a *adapter) SocEstimator() *soc.Estimator {
	return a.Loadpoint.socEstimator
}
