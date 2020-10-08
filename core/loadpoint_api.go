package core

import "github.com/andig/evcc/api"

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

	lp.log.INFO.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.Mode != mode {
		lp.Mode = mode
		lp.publish("mode", mode)
		lp.requestUpdate()
	}
}

// GetTargetSoC returns loadpoint charge target soc
func (lp *LoadPoint) GetTargetSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Target
}

// SetTargetSoC sets loadpoint charge target soc
func (lp *LoadPoint) SetTargetSoC(soc int) error {
	if lp.vehicle == nil {
		return api.ErrNotAvailable
	}

	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Println("set target soc:", soc)

	// apply immediately
	if lp.SoC.Target != soc {
		lp.SoC.Target = soc
		lp.publish("targetSoC", soc)
		lp.requestUpdate()
	}

	return nil
}

// GetMinSoC returns loadpoint charge minimum soc
func (lp *LoadPoint) GetMinSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Min
}

// SetMinSoC sets loadpoint charge minimum soc
func (lp *LoadPoint) SetMinSoC(soc int) error {
	if lp.vehicle == nil {
		return api.ErrNotAvailable
	}

	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Println("set min soc:", soc)

	// apply immediately
	if lp.SoC.Min != soc {
		lp.SoC.Min = soc
		lp.publish("minSoC", soc)
		lp.requestUpdate()
	}

	return nil
}
