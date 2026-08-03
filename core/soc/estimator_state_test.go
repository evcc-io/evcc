package soc

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newAnchorTestEstimator builds an estimator for an 8.5 kWh vehicle, which at
// the default 85% efficiency yields a 10 kWh virtual capacity and 100 Wh per
// soc point — round numbers that keep the expectations readable.
func newAnchorTestEstimator(t *testing.T, capacity float64) *Estimator {
	t.Helper()

	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(capacity).AnyTimes()

	return NewEstimator(util.NewLogger("foo"), api.NewMockCharger(ctrl), vehicle)
}

func TestAnchorReportsReferencePoint(t *testing.T) {
	ce := newAnchorTestEstimator(t, 8.5)

	source := 20.0
	ce.Soc(&source, 0)   // rebase branch, anchors prevSoc at 20
	ce.Soc(&source, 500) // estimating branch

	a := ce.Anchor()
	assert.Equal(t, 20.0, a.Soc)
	assert.Equal(t, 500.0, a.EnergySince)
}

func TestRestoreSeedsEstimate(t *testing.T) {
	ce := newAnchorTestEstimator(t, 8.5)

	// 500 Wh above a 15% anchor is 5 points at 100 Wh per point
	ce.Restore(Anchor{Soc: 15, EnergySince: 500}, 0)

	assert.Equal(t, 20.0, ce.vehicleSoc)
	assert.Equal(t, 15.0, ce.prevSoc, "prevSoc must be seeded with the anchor")
}

// TestRestoreFirstPollDoesNotRebase is the reason Restore seeds prevSoc at all.
// A fresh estimator has prevSoc 0, so without that the first poll produces
// socDelta != 0, takes the rebase branch and discards the restored anchor —
// the estimate would silently fall back to the vehicle's stale reading.
func TestRestoreFirstPollDoesNotRebase(t *testing.T) {
	ce := newAnchorTestEstimator(t, 8.5)
	ce.Restore(Anchor{Soc: 15, EnergySince: 500}, 0)

	source := 15.0
	assert.Equal(t, 20.0, ce.Soc(&source, 0), "restored estimate must survive the first poll")
	assert.InDelta(t, 22.0, ce.Soc(&source, 200), 0.001, "and keep accumulating")
}

// A real reading always wins: once the vehicle reports something other than
// the anchor, the estimator's own rebase branch drops the restored offset. No
// expiry logic is needed for that.
func TestRestoreExpiresOnChangedSourceValue(t *testing.T) {
	ce := newAnchorTestEstimator(t, 8.5)
	ce.Restore(Anchor{Soc: 15, EnergySince: 500}, 0)

	fresh := 42.0
	assert.Equal(t, 42.0, ce.Soc(&fresh, 0))
	assert.Equal(t, 42.0, ce.Soc(&fresh, 0))
}

// evcc may restart mid-session, so the anchor is relative to whatever the
// session counter happens to hold — not necessarily zero.
func TestRestoreWithRunningSessionEnergy(t *testing.T) {
	ce := newAnchorTestEstimator(t, 8.5)

	ce.Restore(Anchor{Soc: 15, EnergySince: 500}, 800)

	source := 15.0
	assert.Equal(t, 20.0, ce.Soc(&source, 800))
}

// A vehicle without configured capacity has no Wh/% to convert the stored
// energy into an offset. Dividing by that zero would produce +Inf, which
// min(_, 100) turns into a plausible-looking full battery.
func TestRestoreWithoutGradientKeepsAnchor(t *testing.T) {
	ce := newAnchorTestEstimator(t, 0)

	ce.Restore(Anchor{Soc: 15, EnergySince: 500}, 0)

	assert.Equal(t, 15.0, ce.vehicleSoc)
}
