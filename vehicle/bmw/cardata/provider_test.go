package cardata

import (
	"testing"
	"time"

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

	keySocOld := "vehicle.drivetrain.batteryManagement.header"
	keySocNew := "vehicle.powertrain.electric.battery.stateOfCharge.displayed"

	// Case 1: Old key is missing, new key is present
	p.rest = map[string]TelematicData{
		keySocNew: {Value: "80"},
	}
	soc, err := p.Soc()
	require.NoError(t, err)
	require.Equal(t, 80.0, soc)

	// Case 2: Old key is empty (null in JSON), new key is present
	p.rest = map[string]TelematicData{
		keySocOld: {Value: ""},
		keySocNew: {Value: "90"},
	}
	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 90.0, soc)

	// Case 3: Old key is nil in streaming, new key is present
	p.rest = nil
	p.streaming = map[string]StreamingData{
		keySocOld: {Value: nil},
		keySocNew: {Value: 95.0},
	}
	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 95.0, soc)

	// Case 4: Old key is present, new key is also present, old key is preferred
	p.rest = map[string]TelematicData{
		keySocOld: {Value: "42"},
		keySocNew: {Value: "90"},
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

	keyRangeOld := "vehicle.drivetrain.electricEngine.kombiRemainingElectricRange"
	keyRangeNew := "vehicle.drivetrain.lastRemainingRange"

	// Case 1: Old key is missing, new key is present
	p.rest = map[string]TelematicData{
		keyRangeNew: {Value: "200"},
	}
	rng, err := p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(200), rng)

	// Case 2: Old key is empty (null in JSON), new key is present
	p.rest = map[string]TelematicData{
		keyRangeOld: {Value: ""},
		keyRangeNew: {Value: "150"},
	}
	rng, err = p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(150), rng)

	// Case 3: Old key is nil in streaming, new key is present
	p.rest = nil
	p.streaming = map[string]StreamingData{
		keyRangeOld: {Value: nil},
		keyRangeNew: {Value: 120.0},
	}
	rng, err = p.Range()
	require.NoError(t, err)
	require.Equal(t, int64(120), rng)

	// Case 4: Old key is present, new key is also present, old key is preferred
	p.rest = map[string]TelematicData{
		keyRangeOld: {Value: "300"},
		keyRangeNew: {Value: "150"},
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
