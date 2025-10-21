package cardata

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCardataStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	p := NewProvider(ctx, util.NewLogger("foo"), nil, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "at",
	}), "client", "vin")

	keySoc := "vehicle.drivetrain.batteryManagement.header"
	p.initial = map[string]TelematicDataPoint{
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
