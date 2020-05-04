package core

type ramp struct {
	*LoadPoint
	targetCurrent          int64
	MinCurrent, MaxCurrent int64
}

// rampUpDown moves stepwise towards target current
func (lp *ramp) rampUpDown(target int64) error {
	current := lp.targetCurrent
	if current == target {
		return nil
	}

	var step int64
	if current < target {
		step = min(current+lp.Sensitivity, target)
	} else if current > target {
		step = max(current-lp.Sensitivity, target)
	}

	step = clamp(step, lp.MinCurrent, lp.MaxCurrent)

	return lp.setTargetCurrent(step)
}

// rampOff disables charger after setting minCurrent. If already disables, this is a nop.
func (lp *ramp) rampOff() error {
	if lp.enabled {
		if lp.targetCurrent == lp.MinCurrent {
			return lp.chargerEnable(false)
		}

		return lp.setTargetCurrent(lp.MinCurrent)
	}

	return nil
}

// rampOn enables charger after setting minCurrent. If already enabled, target will be set.
func (lp *ramp) rampOn(target int64) error {
	if !lp.enabled {
		if err := lp.setTargetCurrent(lp.MinCurrent); err != nil {
			return err
		}

		return lp.chargerEnable(true)
	}

	return lp.setTargetCurrent(target)
}
