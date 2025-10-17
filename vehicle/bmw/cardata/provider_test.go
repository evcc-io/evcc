package cardata

import (
	"context"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type ts struct{}

func (ts *ts) Token() (*oauth2.Token, error) {
	t := &oauth2.Token{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(time.Hour),
	}
	return t, nil
}

func TestMqtt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	p := NewProvider(ctx, util.NewLogger("foo"), nil, new(ts), "client", "vin")
	defer cancel()

	keySoc := "vehicle.drivetrain.batteryManagement.header"
	p.initial = map[string]TelematicDataPoint{
		keySoc: {Value: "42"},
	}

	soc, err := p.Soc()
	require.NoError(t, err)
	require.Equal(t, 42.0, soc)

	mqtt := mqttConnections["client"]
	dataC := mqtt.subscriptions["vin"]
	dataC <- StreamingMessage{
		Vin: "vin",
		Data: map[string]StreamingData{
			keySoc: {Value: "47"},
		},
	}

	time.Sleep(time.Second)

	soc, err = p.Soc()
	require.NoError(t, err)
	require.Equal(t, 47.0, soc)
}
