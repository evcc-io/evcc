package vehicle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bmw/cardata"
)

// Cardata is an api.Vehicle implementation for BMW and Mini cars
type Cardata struct {
	*embed
	*cardata.Provider // provides the api implementations
}

func init() {
	registry.AddCtx("cardata", NewCardataFromConfig)
}

// NewCardataFromConfig creates a new BMW/Mini CarData vehicle
func NewCardataFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration // 50 requests per day
		Stream        string
		mqtt.Config   `mapstructure:",squash"`
		Topic         string
	}{
		Stream: "bmw",
		Topic:  "bmw/raw/+",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// clientid is the BMW client id, not the mqtt one
	cc.Config.ClientID = ""

	if cc.Stream != "bmw" && cc.Stream != "mqtt" {
		return nil, fmt.Errorf("invalid stream: %s", cc.Stream)
	}

	if cc.VIN == "" {
		return nil, errors.New("missing vin")
	}
	cc.VIN = strings.ToUpper(cc.VIN)

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Cache == 0 {
		// for non-streaming use 15m, access controlled by loadpoint
		isStreaming := slices.Contains(cc.embed.Features(), api.Streaming)
		cc.Cache = map[bool]time.Duration{false: 15 * time.Minute, true: 30 * time.Minute}[isStreaming]
	}

	v := &Cardata{
		embed: &cc.embed,
	}

	log := util.NewLogger("cardata").Redact(cc.ClientID, cc.VIN, cc.Password)

	authCtx := util.WithLogger(context.Background(), log)
	ts, err := cardata.NewOAuth(authCtx, cc.ClientID, cc.embed.GetTitle())
	if err != nil {
		return nil, err
	}

	api := cardata.NewAPI(log, ts)

	var conn *cardata.MqttConnector
	if cc.Stream == "mqtt" {
		conn = cardata.NewBrokerConnector(context.Background(), log, cc.Config, cc.Topic)
	} else {
		conn = cardata.NewMqttConnector(context.Background(), log, cc.ClientID, ts)
	}

	v.Provider = cardata.NewProvider(ctx, log, api, ts, conn, cc.VIN, cc.Cache)

	return v, nil
}
