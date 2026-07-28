package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	site "github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// stubSite is a minimal implementation of site.API for testing
type stubSite struct {
	site.API
	bufferSoc float64
}

func (s *stubSite) GetBufferSoc() float64 {
	return s.bufferSoc
}

func TestMinPVRespectsBufferSoc(t *testing.T) {
	Voltage = 230

	ctrl := gomock.NewController(t)
	clk := clock.NewMock()

	tc := []struct {
		name            string
		bufferSoc       float64
		batteryBuffered bool
		sitePower       float64
		expected        float64
	}{
		{
			"no bufferSoc configured, no surplus - charges at min",
			0, false, 500,
			6,
		},
		{
			"bufferSoc configured, battery above (buffered) - charges at min",
			50, true, 500,
			6,
		},
		{
			"bufferSoc configured, battery below (not buffered), no surplus - stops",
			50, false, 500,
			0,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			charger := api.NewMockCharger(ctrl)

			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				clock:          clk,
				charger:        charger,
				minCurrent:     6,
				maxCurrent:     16,
				phases:         1,
				measuredPhases: 1,
				site:           &stubSite{bufferSoc: tc.bufferSoc},
				Enable: loadpoint.ThresholdConfig{
					Delay: 0,
				},
				Disable: loadpoint.ThresholdConfig{
					Delay: 0,
				},
			}

			lp.status = api.StatusC
			lp.enabled = true

			current := lp.pvMaxCurrent(api.ModeMinPV, tc.sitePower, 0, tc.batteryBuffered, false)
			assert.Equal(t, tc.expected, current, tc.name)
		})
	}
}

func TestBufferSocHoldActive(t *testing.T) {
	tc := []struct {
		name       string
		noBattery  bool
		bufferSoc  float64
		batterySoc float64
		charging   bool
		planActive bool
		expected   bool
	}{
		{"no battery configured", true, 65, 0, true, true, false},
		{"no bufferSoc configured", false, 0, 40, true, true, false},
		{"battery above bufferSoc", false, 65, 67, true, true, false},
		{"battery below, plan active", false, 65, 63, true, true, true},
		{"battery below, not charging", false, 65, 63, false, true, false},
		{"battery below, charging but no plan", false, 65, 63, true, false, false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			charger := api.NewMockCharger(ctrl)

			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.charger = charger
			lp.planActive = tc.planActive
			if tc.charging {
				lp.status = api.StatusC
			} else {
				lp.status = api.StatusB
			}

			site := &Site{
				log: util.NewLogger("foo"),
			}
			site.bufferSoc = tc.bufferSoc
			site.battery.Soc = tc.batterySoc
			site.loadpoints = []*Loadpoint{lp}
			if !tc.noBattery {
				site.batteryMeters = []config.Device[api.Meter]{nil}
			}

			assert.Equal(t, tc.expected, site.bufferSocHoldActive(), tc.name)
		})
	}
}
