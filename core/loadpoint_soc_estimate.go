package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/server/db/settings"
)

const (
	// socEstimateMaxAge discards a record that has not been updated recently.
	// It guards against evcc having been down long enough for the world to have
	// moved on, not against the anchor itself being old: while evcc runs, the
	// energy since the anchor is re-derived from the charger's counter on every
	// poll, so an old anchor is not by itself a reason to distrust the record.
	socEstimateMaxAge = 24 * time.Hour

	// socEstimateMaxOffset caps a restored offset, in percentage points
	socEstimateMaxOffset = 50.0
)

// socEstimate is the persisted state of a vehicle's soc estimation.
//
// Kept per vehicle rather than per loadpoint because that is what it
// describes: a state of charge belongs to a battery, and the coordinator
// already moves vehicles between loadpoints. Per loadpoint, two vehicles
// alternating at one charger would overwrite each other's estimate and a
// vehicle moved to a second charger would arrive without one.
type socEstimate struct {
	AnchorSoc         float64   `json:"anchorSoc"`         // last soc the vehicle reported, in %
	EnergySinceAnchor float64   `json:"energySinceAnchor"` // energy delivered at the charger since, in Wh
	Updated           time.Time `json:"updated"`
}

// offset returns the estimate's distance above the anchor, in percentage
// points. Derived from energy rather than from the estimate, which is clamped
// at 100 and would hide an implausibly large offset.
func (se socEstimate) offset(energyPerSocStep float64) float64 {
	if energyPerSocStep <= 0 {
		return 0
	}
	return se.EnergySinceAnchor / energyPerSocStep
}

// plausible reports whether the record may be restored.
//
// There is deliberately no check against the vehicle's current reading here:
// at restore time none has been fetched yet, and none is needed. Restore()
// seeds prevSoc with the anchor, so a first poll that reads a changed value
// takes the estimator's rebase branch and drops the offset by itself.
func (se socEstimate) plausible(energyPerSocStep float64, now time.Time) bool {
	switch {
	case se.EnergySinceAnchor <= 0:
		return false
	case now.Sub(se.Updated) > socEstimateMaxAge:
		return false
	case se.offset(energyPerSocStep) > socEstimateMaxOffset:
		return false
	default:
		return true
	}
}

func socEstimateKey(name string) string {
	return fmt.Sprintf("vehicle.%s.%s", name, keys.SocEstimate)
}

func loadSocEstimate(name string) (socEstimate, bool) {
	var se socEstimate
	if err := settings.Json(socEstimateKey(name), &se); err != nil {
		return socEstimate{}, false
	}
	return se, true
}

// saveSocEstimate stores the record. It only reaches the database when evcc's
// settings ticker flushes dirty entries, which it does every minute and on
// shutdown.
func saveSocEstimate(name string, se socEstimate) error {
	return settings.SetJson(socEstimateKey(name), se)
}

// updateSocEstimate mirrors the running estimator into the persisted record.
//
// estimator and vehicleName are the caller's snapshot rather than fields read
// from lp: setActiveVehicle runs synchronously on the http and mqtt
// goroutines, so re-reading either here could pair one vehicle's estimator
// with another's name and write the first's energy into the second's record —
// both records staying syntactically valid. Same reason publishSocAndRange
// snapshots socEstimator, see #16180.
func (lp *Loadpoint) updateSocEstimate(estimator *soc.Estimator, vehicleName string) {
	if estimator == nil || vehicleName == "" {
		return
	}

	a := estimator.Anchor()
	if a.EnergySince <= 0 {
		return
	}

	if err := saveSocEstimate(vehicleName, socEstimate{
		AnchorSoc:         a.Soc,
		EnergySinceAnchor: a.EnergySince,
		Updated:           lp.clock.Now(),
	}); err != nil {
		lp.log.ERROR.Printf("soc estimate: %v", err)
	}
}

// restoreSocEstimate seeds the estimator from the persisted record.
//
// Called from createSession, deliberately not from setActiveVehicle: the
// anchor is measured against the session energy counter, and
// evVehicleConnectHandler runs vehicleDefaultOrDetect — and with it
// setActiveVehicle — before createSession resets that counter. Anchoring
// before the reset yields an anchor relative to the previous session, which
// looks correct at process start (the counter really is zero there) but
// collapses the estimate to the vehicle's stale reading on every subsequent
// connect. splitSession calls createSession too, so both paths are covered.
func (lp *Loadpoint) restoreSocEstimate() {
	estimator := lp.socEstimator
	if estimator == nil || lp.socEstimateVehicle == "" {
		return
	}

	se, ok := loadSocEstimate(lp.socEstimateVehicle)
	if !ok {
		return
	}

	if !se.plausible(estimator.EnergyPerSocStep(), lp.clock.Now()) {
		return
	}

	estimator.Restore(soc.Anchor{
		Soc:         se.AnchorSoc,
		EnergySince: se.EnergySinceAnchor,
	}, lp.GetChargedEnergy())

	lp.log.DEBUG.Printf("soc estimate restored: anchor %.1f%%, %.0fWh since", se.AnchorSoc, se.EnergySinceAnchor)
}
