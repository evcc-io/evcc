package cardata

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCardataStreaming(t *testing.T) {
	ctx := t.Context()

	p := NewProvider(ctx, util.NewLogger("foo"), nil, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "at",
	}), "client", "vin", 0)

	// prevent container panic
	p.updated = time.Now()

	keySoc := "vehicle.drivetrain.batteryManagement.header"
	p.rest = map[string]TelematicData{
		keySoc: {Value: "42"},
	}

	soc, err := p.Soc()
	require.NoError(t, err)
	require.Equal(t, 42.0, soc)

	mqtt := mqttConnections["client"]
	dataC := mqtt.subscriptions["vin"]
	require.NotNil(t, dataC, "streaming channel")

	dataC <- StreamingMessage{
		Vin: "vin",
		Data: map[string]StreamingData{
			keySoc: {Value: "47"},
		},
	}

	// process first message
	dataC <- StreamingMessage{}
	dataC <- StreamingMessage{}

	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 47.0, soc)
}

func TestSocFallback(t *testing.T) {
	ctx := t.Context()

	p := NewProvider(ctx, util.NewLogger("foo"), nil, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "at",
	}), "client", "vin", 0)

	// prevent container panic
	p.updated = time.Now()

	keySocPrimary := "vehicle.drivetrain.batteryManagement.header"
	keySocFallback := "vehicle.powertrain.electric.battery.stateOfCharge.displayed"

	// Case 1: Primary key is missing, fallback key is present
	p.rest = map[string]TelematicData{
		keySocFallback: {Value: "80"},
	}
	soc, err := p.Soc()
	require.NoError(t, err)
	require.Equal(t, 80.0, soc)

	// Case 2: Primary key is empty (null in JSON), fallback key is present
	p.rest = map[string]TelematicData{
		keySocPrimary:  {Value: ""},
		keySocFallback: {Value: "90"},
	}
	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 90.0, soc)

	// Case 3: Primary key is nil in streaming, fallback key is present
	p.rest = nil
	p.streaming = map[string]StreamingData{
		keySocPrimary:  {Value: nil},
		keySocFallback: {Value: 95.0},
	}
	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 95.0, soc)

	// Case 4: Primary key is present, fallback key is also present, primary key is preferred
	p.rest = map[string]TelematicData{
		keySocPrimary:  {Value: "42"},
		keySocFallback: {Value: "90"},
	}
	p.streaming = nil
	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 42.0, soc)

	// Case 5: Both keys are absent, returns error
	p.rest = map[string]TelematicData{}
	soc, err = p.Soc()
	require.Error(t, err)
	require.Equal(t, 0.0, soc)
}

func TestRangeFallback(t *testing.T) {
	ctx := t.Context()

	p := NewProvider(ctx, util.NewLogger("foo"), nil, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "at",
	}), "client", "vin", 0)

	// prevent container panic
	p.updated = time.Now()

	keyRangePrimary := "vehicle.drivetrain.electricEngine.kombiRemainingElectricRange"
	keyRangeFallback := "vehicle.drivetrain.lastRemainingRange"

	// Case 1: Primary key is missing, fallback key is present
	p.rest = map[string]TelematicData{
		keyRangeFallback: {Value: "200"},
	}
	rng, err := p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(200), rng)

	// Case 2: Primary key is empty (null in JSON), fallback key is present
	p.rest = map[string]TelematicData{
		keyRangePrimary:  {Value: ""},
		keyRangeFallback: {Value: "150"},
	}
	rng, err = p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(150), rng)

	// Case 3: Primary key is nil in streaming, fallback key is present
	p.rest = nil
	p.streaming = map[string]StreamingData{
		keyRangePrimary:  {Value: nil},
		keyRangeFallback: {Value: 120.0},
	}
	rng, err = p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(120), rng)

	// Case 4: Primary key is present, fallback key is also present, primary key is preferred
	p.rest = map[string]TelematicData{
		keyRangePrimary:  {Value: "300"},
		keyRangeFallback: {Value: "150"},
	}
	p.streaming = nil
	rng, err = p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(300), rng)

	// Case 5: Both keys are absent, returns error
	p.rest = map[string]TelematicData{}
	rng, err = p.Range()
	require.Error(t, err)
	require.Equal(t, int64(0), rng)
}

func TestStatusFallback(t *testing.T) {
	ctx := t.Context()

	p := NewProvider(ctx, util.NewLogger("foo"), nil, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "at",
	}), "client", "vin", 0)

	// prevent container panic
	p.updated = time.Now()

	keyChargingStatusPrimary := "vehicle.drivetrain.electricEngine.charging.status"
	keyChargingStatusFallback := "vehicle.drivetrain.electricEngine.charging.hvStatus"
	keyPortStatusPrimary := "vehicle.body.chargingPort.status"
	keyPortStatusFallback := "vehicle.body.chargingPort.combinedStatus"

	// Case 1: Charging status fallback works
	p.rest = map[string]TelematicData{
		keyChargingStatusFallback: {Value: "CHARGING"},
	}
	status, err := p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusC, status)

	// Case 2: charging status primary is present, fallback is present, primary is preferred
	p.rest = map[string]TelematicData{
		keyChargingStatusPrimary:  {Value: "CHARGINGACTIVE"},
		keyChargingStatusFallback: {Value: "WAITING_FOR_CHARGING"}, // would be StatusB
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusC, status)

	// Case 3: charging status empty, port primary is missing, port fallback is present -> StatusB
	p.rest = map[string]TelematicData{
		keyPortStatusFallback: {Value: "CONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// Case 4: charging status empty, port primary is empty, port fallback is present -> StatusA (disconnected)
	p.rest = map[string]TelematicData{
		keyPortStatusPrimary:  {Value: ""},
		keyPortStatusFallback: {Value: "DISCONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)

	// Case 5: charging status empty, port primary nil in streaming, port fallback is present -> StatusB
	p.rest = nil
	p.streaming = map[string]StreamingData{
		keyPortStatusPrimary:  {Value: nil},
		keyPortStatusFallback: {Value: "CONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// Case 6: port primary is present, fallback is also present, primary is preferred
	p.rest = map[string]TelematicData{
		keyPortStatusPrimary:  {Value: "CONNECTED"},
		keyPortStatusFallback: {Value: "DISCONNECTED"},
	}
	p.streaming = nil
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// Case 7: Both port keys are absent, returns error
	p.rest = map[string]TelematicData{}
	status, err = p.Status()
	require.Error(t, err)
	require.Equal(t, api.ErrNotAvailable, err)
	require.Equal(t, api.StatusNone, status)

	// Case 8: charging status state takes precedence over port state (charging status = C, port = DISCONNECTED)
	p.rest = map[string]TelematicData{
		keyChargingStatusPrimary: {Value: "CHARGINGACTIVE"},
		keyPortStatusPrimary:     {Value: "DISCONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusC, status)

	// Case 9: charging status state takes precedence over port state (charging status = B, port = DISCONNECTED)
	p.rest = map[string]TelematicData{
		keyChargingStatusPrimary: {Value: "INITIALIZATION"},
		keyPortStatusPrimary:     {Value: "DISCONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// Case 10: charging status state is "NOCHARGING", which allows no distinction between charging status A and B, so fall back to port state
	p.rest = map[string]TelematicData{
		keyChargingStatusPrimary: {Value: "NOCHARGING"},
		keyPortStatusPrimary:     {Value: "CONNECTED"},
	}
	status, err = p.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)
}
