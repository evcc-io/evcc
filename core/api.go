package core

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
)

// Param is the broadcast channel data type
type Param struct {
	LoadPoint string
	Key       string
	Val       interface{}
}

// Configuration is the loadpoint feature structure
type Configuration struct {
	Mode        string `json:"mode"`
	Phases      int64  `json:"phases"`
	MinCurrent  int64  `json:"minCurrent"`
	MaxCurrent  int64  `json:"maxCurrent"`
	GridMeter   bool   `json:"gridMeter"`
	PVMeter     bool   `json:"pvMeter"`
	ChargeMeter bool   `json:"chargeMeter"`
	SoC         bool   `json:"soc"`
	SoCCapacity int64  `json:"socCapacity"`
	SoCTitle    string `json:"socTitle"`
}

// Configuration returns meter configuration
func (lp *LoadPoint) Configuration() Configuration {
	c := Configuration{
		Mode:        string(lp.GetMode()),
		Phases:      lp.Phases,
		MinCurrent:  lp.MinCurrent,
		MaxCurrent:  lp.MaxCurrent,
		GridMeter:   lp.GridMeter != nil,
		PVMeter:     lp.PVMeter != nil,
		ChargeMeter: lp.hasChargeMeter(),
	}

	if lp.SoC != nil {
		c.SoC = true
		if soc, ok := lp.SoC.(*SoC); ok {
			c.SoCCapacity = soc.Capacity
			c.SoCTitle = soc.Title
		}
	}

	return c
}

func (lp *LoadPoint) hasChargeMeter() bool {
	_, isWrapped := lp.ChargeMeter.(*wrapper.ChargeMeter)
	return lp.ChargeMeter != nil && !isWrapped
}

// Dump loadpoint configuration
func (lp *LoadPoint) Dump() {
	soc := lp.SoC != nil
	grid := lp.GridMeter != nil
	pv := lp.PVMeter != nil
	log.INFO.Printf("%s config: soc %s grid %s pv %s charge %s", lp.Name,
		presence[soc],
		presence[grid],
		presence[pv],
		presence[lp.hasChargeMeter()],
	)
	log.INFO.Printf("%s charge mode: %s", lp.Name, lp.GetMode())
}

// Update triggers loadpoint to run main control loop and push messages to UI socket
func (lp *LoadPoint) Update() {
	select {
	case lp.triggerChan <- struct{}{}: // non-blocking send
	default:
	}
}

// GetMode returns loadpoint charge mode
func (lp *LoadPoint) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()

	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *LoadPoint) SetMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	log.INFO.Printf("%s set charge mode: %s", lp.Name, string(mode))
	lp.Mode = mode

	lp.Update()
}

// chargeDuration returns for how long the charge cycle has been running
func (lp *LoadPoint) chargeDuration() time.Duration {
	d, err := lp.ChargeTimer.ChargingTime()
	if err != nil {
		log.ERROR.Printf("%s charge timer error: %v", lp.Name, err)
	}
	return d
}

// chargedEnergy returns energy consumption since charge start in kWh
func (lp *LoadPoint) chargedEnergy() float64 {
	f, err := lp.ChargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s charge rater error: %v", lp.Name, err)
	}
	return f
}

// remainingChargeDuration returns the remaining charge time
func (lp *LoadPoint) remainingChargeDuration(chargePercent float64) time.Duration {
	if !lp.charging {
		return -1
	}

	if lp.chargePower > 0 {
		if soc, ok := lp.SoC.(*SoC); ok {
			whRemaining := (1 - chargePercent/100.0) * 1e3 * float64(soc.Capacity)
			return time.Duration(float64(time.Hour) * whRemaining / lp.chargePower)
		}
	}

	return -1
}

// publish state of charge and remaining charge duration
func (lp *LoadPoint) publishSoC() {
	if lp.SoC == nil {
		return
	}

	if lp.connected() {
		f, err := lp.SoC.ChargeState()
		if err == nil {
			log.DEBUG.Printf("%s soc charge: %.1f%%", lp.Name, f)
			lp.publish("socCharge", f)
			lp.publish("chargeEstimate", lp.remainingChargeDuration(f))
			return
		}
		log.ERROR.Printf("%s soc error: %v", lp.Name, err)
	}

	lp.publish("socCharge", "—")
	lp.publish("chargeEstimate", -1)
}
