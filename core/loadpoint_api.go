package core

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/andig/evcc/api"
)

// Param is the broadcast channel data type
type Param struct {
	Key string
	Val interface{}
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
		Mode:        string(lp.state.Mode()),
		Phases:      lp.Phases,
		MinCurrent:  lp.MinCurrent,
		MaxCurrent:  lp.MaxCurrent,
		GridMeter:   lp.GridMeter != nil,
		PVMeter:     lp.PVMeter != nil,
		ChargeMeter: lp.chargeMeterPresent(),
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

// ChargeMode updates charge mode
func (lp *LoadPoint) ChargeMode(mode api.ChargeMode) error {
	// pv modes require GridMeter
	if mode == api.ModeMinPV || mode == api.ModePV {
		// check if charger is controllable
		_, chargerControllable := lp.Charger.(api.ChargeController)

		if (lp.PVMeter == nil && lp.GridMeter == nil) || !chargerControllable {
			return errors.New("invalid charge mode: " + string(mode))
		}
	}

	if lp.state.Mode() != mode {
		// start cycle detection async from http call
		go lp.chargingCycle(mode != api.ModeOff)

		// update mode and enable/disable charger
		lp.chargeMode(mode)
		return lp.chargerEnable(mode != api.ModeOff)
	}

	return nil
}

// ChargeDuration returns for how long the charge cycle has been running
func (lp *LoadPoint) ChargeDuration() time.Duration {
	d, err := lp.ChargeTimer.ChargingTime()
	if err != nil {
		log.ERROR.Printf("%s charge timer error: %v", lp.Name, err)
	}
	return d
}

// ChargedEnergy returns energy consumption since charge start in Wh
func (lp *LoadPoint) ChargedEnergy() float64 {
	f, err := lp.ChargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s charge rater error: %v", lp.Name, err)
	}
	return f
}

// Connected returns the EVs connection state
func (lp *LoadPoint) Connected() bool {
	status := lp.state.Status()
	return status == api.StatusB || status == api.StatusC
}

// SoCChargeState returns the soc battery charge state
func (lp *LoadPoint) SoCChargeState() float64 {
	if !lp.Connected() {
		return math.NaN()
	}

	f, err := lp.SoC.ChargeState()
	if err != nil {
		log.ERROR.Printf("%s soc error: %v", lp.Name, err)
		return math.NaN()
	}

	lp.state.SetSocCharge(f)

	return f
}

// ChargeRemainingDuration returns the remaining charge time
func (lp *LoadPoint) ChargeRemainingDuration() time.Duration {
	if !lp.state.Charging() {
		return -1
	}

	if currentPower := lp.state.ChargePower(); currentPower > 0 {
		if soc, ok := lp.SoC.(*SoC); ok {
			whRemaining := (1 - lp.state.SocCharge()/100.0) * 1000.0 * float64(soc.Capacity)
			return time.Duration(float64(time.Hour) * whRemaining / currentPower)
		}
	}

	return -1
}

func (lp *LoadPoint) publishMeter(name string, meter api.Meter, clientPush chan<- Param) {
	if f, err := meter.CurrentPower(); err == nil {
		key := name + "Power"
		log.TRACE.Printf("%s %s power: %.1fW", lp.Name, name, f)
		clientPush <- Param{Key: key, Val: f}
	} else {
		log.ERROR.Printf("%s %v", lp.Name, err)
	}
}

func (lp *LoadPoint) publishCharging(clientPush chan<- Param) {
	if lp.state.Charging() {
		if f, err := lp.Charger.ActualCurrent(); err == nil {
			log.TRACE.Printf("%s charge current: %dA", lp.Name, f)
			clientPush <- Param{Key: "chargeCurrent", Val: f}
		} else {
			log.ERROR.Printf("%s update charger current failed: %v", lp.Name, err)
		}
	} else {
		clientPush <- Param{Key: "chargeCurrent", Val: 0.0}
	}
}

func (lp *LoadPoint) publishSoC(clientPush chan<- Param) {
	f := lp.SoCChargeState()
	if math.IsNaN(f) {
		log.TRACE.Printf("%s soc charge: —", lp.Name)
		clientPush <- Param{Key: "socCharge", Val: "—"}
	} else {
		log.TRACE.Printf("%s soc charge: %.1f%%", lp.Name, f)
		clientPush <- Param{Key: "socCharge", Val: f}
	}

	clientPush <- Param{Key: "chargeEstimate", Val: lp.ChargeRemainingDuration()}
}

// Publish publishes loadpoint data to broadcast channel
func (lp *LoadPoint) Publish(ctx context.Context, clientPush chan<- Param) {
	clientPush <- Param{Key: "mode", Val: string(lp.state.Mode())}
	clientPush <- Param{Key: "connected", Val: lp.Connected()}
	clientPush <- Param{Key: "charging", Val: lp.state.Charging()}

	var wg sync.WaitGroup

	for name, meter := range map[string]api.Meter{
		"grid":   lp.GridMeter,
		"pv":     lp.PVMeter,
		"charge": lp.ChargeMeter,
	} {
		if meter != nil {
			wg.Add(1)
			go func(name string, meter api.Meter) {
				lp.publishMeter(name, meter, clientPush)
				wg.Done()
			}(name, meter)
		}
	}

	wg.Add(1)
	go func() {
		lp.publishCharging(clientPush)
		wg.Done()
	}()

	if lp.SoC != nil {
		wg.Add(1)
		go func() {
			lp.publishSoC(clientPush)
			wg.Done()
		}()
	}

	wg.Wait()

	clientPush <- Param{Key: "chargedEnergy", Val: lp.ChargedEnergy()}
	clientPush <- Param{Key: "chargeDuration", Val: lp.ChargeDuration()}
}
