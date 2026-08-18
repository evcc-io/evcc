package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/plugchoice"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPlugchoiceFixture creates a charger backed by static API responses
func newPlugchoiceFixture(conn plugchoice.Connector, power plugchoice.PowerResponse) *Plugchoice {
	conn.ConnectorID = 1

	c := &Plugchoice{connector: 1, current: 6}

	c.statusG = util.ResettableCached(func() (plugchoice.StatusResponse, error) {
		return plugchoice.StatusResponse{
			Data: plugchoice.ChargerData{Connectors: []plugchoice.Connector{conn}},
		}, nil
	}, 0)

	c.powerG = util.ResettableCached(func() (plugchoice.PowerResponse, error) {
		return power, nil
	}, 0)

	return c
}

// TestPlugchoiceMetering verifies that measurements follow the connector status rather
// than the locally cached enable state (https://github.com/evcc-io/evcc/issues/32419)
func TestPlugchoiceMetering(t *testing.T) {
	// the API only receives meter values during a transaction and may keep serving
	// the last known sample afterwards
	stale := plugchoice.PowerResponse{KW: "7.4", L1: "10.7", L2: "10.7", L3: "10.7"}

	tc := []struct {
		name     string
		status   core.ChargePointStatus
		power    plugchoice.PowerResponse
		expPower float64
		expL1    float64
	}{
		{"charging", core.ChargePointStatusCharging, stale, 7400, 10.7},
		// session started outside of evcc, so Enable() was never called
		{"charging, single phase", core.ChargePointStatusCharging,
			plugchoice.PowerResponse{KW: "3.6", L1: "16.0", L2: "-", L3: " 0.0"}, 3600, 16},
		// not charging- stale values must not be reported
		{"suspended by ev", core.ChargePointStatusSuspendedEV, stale, 0, 0},
		{"preparing", core.ChargePointStatusPreparing, stale, 0, 0},
		{"available", core.ChargePointStatusAvailable, stale, 0, 0},
		{"finishing", core.ChargePointStatusFinishing, stale, 0, 0},
		{"suspended by evse", core.ChargePointStatusSuspendedEVSE, stale, 0, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			c := newPlugchoiceFixture(plugchoice.Connector{Status: tc.status}, tc.power)

			power, err := c.CurrentPower()
			require.NoError(t, err)
			assert.Equal(t, tc.expPower, power, "power")

			l1, _, _, err := c.Currents()
			require.NoError(t, err)
			assert.Equal(t, tc.expL1, l1, "l1")
		})
	}
}

func TestPlugchoiceEnabled(t *testing.T) {
	tc := []struct {
		name     string
		status   core.ChargePointStatus
		limit    *int
		cached   bool // locally cached enable state
		expected bool
	}{
		// conclusive status wins- covers sessions started outside of evcc
		{"charging", core.ChargePointStatusCharging, lo.ToPtr(0), false, true},
		{"suspended by ev", core.ChargePointStatusSuspendedEV, nil, false, true},
		{"suspended by evse", core.ChargePointStatusSuspendedEVSE, lo.ToPtr(16), true, false},
		// inconclusive status- applied limit beats the locally cached state
		{"preparing, limit applied", core.ChargePointStatusPreparing, lo.ToPtr(16), false, true},
		{"preparing, limit zero", core.ChargePointStatusPreparing, lo.ToPtr(0), true, false},
		{"available, limit applied", core.ChargePointStatusAvailable, lo.ToPtr(6), false, true},
		// API does not report the limit- fall back to the cached state
		{"preparing, no limit reported", core.ChargePointStatusPreparing, nil, true, true},
		{"available, no limit reported", core.ChargePointStatusAvailable, nil, false, false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			c := newPlugchoiceFixture(plugchoice.Connector{
				Status:       tc.status,
				CurrentLimit: tc.limit,
			}, plugchoice.PowerResponse{})
			c.enabled = tc.cached

			enabled, err := c.Enabled()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, enabled)
		})
	}
}

func TestPlugchoiceGetMaxCurrent(t *testing.T) {
	c := newPlugchoiceFixture(plugchoice.Connector{CurrentLimit: lo.ToPtr(16)}, plugchoice.PowerResponse{})
	current, err := c.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 16.0, current)

	c = newPlugchoiceFixture(plugchoice.Connector{}, plugchoice.PowerResponse{})
	_, err = c.GetMaxCurrent()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestPlugchoiceConnectorNotFound(t *testing.T) {
	c := newPlugchoiceFixture(plugchoice.Connector{Status: core.ChargePointStatusCharging}, plugchoice.PowerResponse{})
	c.connector = 2

	_, err := c.Status()
	assert.Error(t, err)

	// measurements must not be reported when the connector is unknown
	power, err := c.CurrentPower()
	assert.Error(t, err)
	assert.Zero(t, power)
}
