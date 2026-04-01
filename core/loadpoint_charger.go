package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

// chargerHasFeature checks availability of charger feature
func (lp *Loadpoint) chargerHasFeature(f api.Feature) bool {
	return hasFeature(lp.charger, f)
}

// publishChargerFeature publishes availability of charger features
func (lp *Loadpoint) publishChargerFeature(f api.Feature) {
	lp.publish(keys.ChargerFeature+f.String(), lp.chargerHasFeature(f))
}

// chargerIdentifier returns charger id if available
func (lp *Loadpoint) chargerIdentifier() (string, error) {
	if c, ok := api.Cap[api.Identifier](lp.charger); ok {
		return c.Identify()
	}
	return "", api.ErrNotAvailable
}

