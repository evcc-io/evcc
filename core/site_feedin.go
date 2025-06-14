package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (site *Site) smartFeedinDisableAvailable() bool {
	for _, dev := range site.pvMeters {
		meter := dev.Instance()

		if _, ok := meter.(api.FeedinDisableController); ok {
			return true
		}
	}

	return false
}

func (site *Site) SmartFeedinDisableActive() bool {
	site.RLock()
	defer site.RUnlock()
	return site.smartFeedinDisableActive
}

func (site *Site) UpdateSmartFeedinDisable(rate api.Rate) error {
	limit := site.GetSmartFeedinDisableLimit()
	enable := limit != nil && !rate.IsZero() && rate.Value <= *limit

	if site.SmartFeedinDisableActive() == enable {
		return nil
	}

	err := site.enableSmartFeedinLimit(enable)
	if err == nil {
		site.Lock()
		site.smartFeedinDisableActive = enable
		site.publish(keys.SmartFeedinDisableActive, enable)
		site.Unlock()
	}

	return err
}

func (site *Site) enableSmartFeedinLimit(enable bool) error {
	for _, dev := range site.pvMeters {
		meter := dev.Instance()

		if fl, ok := meter.(api.FeedinDisableController); ok {
			if err := fl.FeedinDisableLimitEnable(enable); err != nil {
				return err
			}
		}
	}

	return nil
}
