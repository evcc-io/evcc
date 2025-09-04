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

		if _, ok := meter.(api.FeedInDisableController); ok {
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

	err := site.setFeedInDisable(enable)
	if err == nil {
		site.Lock()
		site.smartFeedInDisableActive = enable
		site.publish(keys.SmartFeedInDisableActive, enable)
		site.Unlock()
	}

	return err
}

func (site *Site) setFeedInDisable(enable bool) error {
	for _, dev := range site.pvMeters {
		meter := dev.Instance()

		if fl, ok := meter.(api.FeedInDisableController); ok {
			if err := fl.FeedInDisableLimitEnable(enable); err != nil {
				return err
			}
		}
	}

	return nil
}
