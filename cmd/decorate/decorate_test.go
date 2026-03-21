package main

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDecorator(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := api.NewMockCharger(ctrl)

	enG := func() (float64, error) { return 0, nil }
	phS := func(int) error { return nil }
	phG := func() (int, error) { return 0, nil }

	check := func(t *testing.T, result api.Charger, wantEnergy, wantSwitcher, wantGetter bool) {
		t.Helper()
		assert.NotNil(t, result)

		_, ok := api.Cap[api.MeterEnergy](result)
		assert.Equal(t, wantEnergy, ok, "MeterEnergy")

		_, ok = api.Cap[api.PhaseSwitcher](result)
		assert.Equal(t, wantSwitcher, ok, "PhaseSwitcher")

		_, ok = api.Cap[api.PhaseGetter](result)
		assert.Equal(t, wantGetter, ok, "PhaseGetter")
	}

	// dependency rule: PhaseGetter requires PhaseSwitcher
	check(t, decorateTest(c, nil, nil, nil), false, false, false)
	check(t, decorateTest(c, nil, nil, phG), false, false, false) // phG ignored without phS
	check(t, decorateTest(c, nil, phS, nil), false, true, false)
	check(t, decorateTest(c, nil, phS, phG), false, true, true)
	check(t, decorateTest(c, enG, nil, nil), true, false, false)
	check(t, decorateTest(c, enG, nil, phG), true, false, false) // phG ignored without phS
	check(t, decorateTest(c, enG, phS, nil), true, true, false)
	check(t, decorateTest(c, enG, phS, phG), true, true, true)
}
