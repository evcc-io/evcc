package core

import (
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
)

// chargerHasFeature checks availability of charger feature
func (lp *Loadpoint) chargerHasFeature(f api.Feature) bool {
	c, ok := lp.charger.(api.FeatureDescriber)
	return ok && slices.Contains(c.Features(), f)
}

// publishChargerFeature publishes availability of charger features
func (lp *Loadpoint) publishChargerFeature(f api.Feature) {
	lp.publish(keys.ChargerFeature+f.String(), lp.chargerHasFeature(f))
}

// chargerId returns charger id if available
func (lp *Loadpoint) chargerId() (string, error) {
	if c, ok := lp.charger.(api.Identifier); ok {
		return c.Identify()
	}
	return "", api.ErrNotAvailable
}

// chargerSoc returns charger soc if available
func (lp *Loadpoint) chargerSoc() (float64, error) {
	if c, ok := lp.charger.(api.Battery); ok {
		return soc.Guard(c.Soc())
	}
	return 0, api.ErrNotAvailable
}
