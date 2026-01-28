package ecoflow

import (
	"time"

	"github.com/evcc-io/evcc/util"
)

// config for Stream and PowerStream devices
type config struct {
	URI       string
	SN        string
	AccessKey string
	SecretKey string
	Usage     string
	Cache     time.Duration
	// Battery limits (optional, for BatterySocLimiter)
	MinSoc   float64
	MaxSoc   float64
	Capacity float64 // kWh
}

func (c *config) decode(other map[string]any) error {
	c.Cache = 10 * time.Second
	return util.DecodeOther(other, c)
}

// response wrapper for EcoFlow API
type response[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// StreamData - API response data for Stream devices
type StreamData struct {
	Relay2Onoff   bool    `json:"relay2Onoff"`   // AC1 switch
	Relay3Onoff   bool    `json:"relay3Onoff"`   // AC2 switch
	PowGetPvSum   float64 `json:"powGetPvSum"`   // PV power (W)
	PowGetSysGrid float64 `json:"powGetSysGrid"` // Grid power (W)
	PowGetSysLoad float64 `json:"powGetSysLoad"` // Load power (W)
	CmsBattSoc    float64 `json:"cmsBattSoc"`    // Battery SOC (%)
	PowGetBpCms   float64 `json:"powGetBpCms"`   // Battery power (W)
	// Limits from API
	CmsMaxChgSoc int `json:"cmsMaxChgSoc"` // Charge limit (%)
	CmsMinDsgSoc int `json:"cmsMinDsgSoc"` // Discharge limit (%)
}

// PowerStreamData - API response data for PowerStream devices
type PowerStreamData struct {
	Pv1InputWatts  float64 `json:"pv1InputWatts"`  // PV1 power (W)
	Pv2InputWatts  float64 `json:"pv2InputWatts"`  // PV2 power (W)
	BatWatts       float64 `json:"batInputWatts"`  // Battery power (W)
	InvOutputWatts float64 `json:"invOutputWatts"` // Inverter output (W)
	BatSoc         int     `json:"batSoc"`         // Battery SOC (%)
	InvOnOff       int     `json:"invOnOff"`       // Inverter state
	PermanentWatts int     `json:"permanentWatts"` // Custom load (W)
	LowerLimit     int     `json:"lowerLimit"`     // Discharge limit (%)
	UpperLimit     int     `json:"upperLimit"`     // Charge limit (%)
}
