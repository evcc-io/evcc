package core

import "github.com/andig/evcc/api"

// LoadpointAPI is the external loadpoint API
type LoadpointAPI interface {
	GetMode() api.ChargeMode
	SetMode(api.ChargeMode)
	GetTargetSoC() int
	SetTargetSoC(int) error
	GetMinSoC() int
	SetMinSoC(int) error
}

// SiteAPI is the external site API
type SiteAPI interface {
	Healthy() bool
	Configuration() SiteConfiguration
	LoadPoints() []*LoadPoint
	LoadpointAPI
}
