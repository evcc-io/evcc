package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (site *Site) smartFeedInDisableAvailable() bool {
	if !site.isDynamicTariff(api.TariffUsageFeedIn) {
		return false
	}

	for _, dev := range site.pvMeters {
		meter := dev.Instance()

		if _, ok := meter.(api.Curtailer); ok {
			return true
		}
	}

	return false
}

func (site *Site) SmartFeedInDisableActive() bool {
	site.RLock()
	defer site.RUnlock()
	return site.smartFeedInDisableActive
}

func (site *Site) UpdateSmartFeedInDisable(rate api.Rate) error {
	limit := site.GetSmartFeedInDisableLimit()
	enable := limit != nil && !rate.IsZero() && rate.Value <= *limit

	if site.SmartFeedInDisableActive() == enable {
		return nil
	}

	// flag only; curtailPV applies the effective percent every cycle
	site.Lock()
	site.smartFeedInDisableActive = enable
	site.publish(keys.SmartFeedInDisableActive, enable)
	site.Unlock()

	return nil
}

// effectiveCurtailPercent is the strictest curtailment of grid-required (HEMS)
// and smart feed-in disable (0% = fully curtailed). nil = no active limit.
func (site *Site) effectiveCurtailPercent() *int {
	if site.SmartFeedInDisableActive() {
		return new(0)
	}
	if site.hems == nil {
		return nil
	}
	return site.hems.CurtailedPercent()
}

// revertSmartFeedInCurtail drops smart feed-in's curtailment and restores the
// grid-required level (100% = uncurtailed when no grid limit). Used on shutdown.
func (site *Site) revertSmartFeedInCurtail() error {
	if !site.SmartFeedInDisableActive() {
		return nil
	}

	percent := 100
	if site.hems != nil {
		if p := site.hems.CurtailedPercent(); p != nil {
			percent = *p
		}
	}

	return site.curtailPV(&percent)
}
