package vehicle

import (
	"context"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/cloud"
	"github.com/andig/evcc/proto/pb"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// Cloud is an api.Vehicle implementation for Cloud cars
type Cloud struct {
	*embed
	brand        string
	config       map[string]string
	token        string
	client       pb.VehicleClient
	vehicleID    int64
	chargeStateG func() (float64, error)
}

func init() {
	registry.Add("cloud", NewCloudFromConfig)
}

// NewCloudFromConfig creates a new vehicle
func NewCloudFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Token    string
		Title    string
		Capacity int64
		Brand    string
		Other    map[string]string `mapstructure:",remain"`
		Cache    time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("cloud")
	client, err := cloud.Client(log, cloud.Host)

	var vehicleID int64
	if err == nil {
		req := &pb.NewRequest{
			Token:  cc.Token,
			Type:   cc.Brand,
			Config: cc.Other,
		}

		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		var res *pb.NewReply
		if res, err = client.New(ctx, req); err == nil {
			vehicleID = res.VehicleId
		}
	}

	v := &Cloud{
		embed:     &embed{cc.Title, cc.Capacity},
		brand:     cc.Brand,
		config:    cc.Other,
		token:     cc.Token,
		client:    client,
		vehicleID: vehicleID,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, err
}

// chargeState implements the api.Vehicle interface
func (v *Cloud) chargeState() (float64, error) {
	req := &pb.SoCRequest{
		Token:     v.token,
		VehicleId: v.vehicleID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	res, err := v.client.SoC(ctx, req)
	return res.GetSoc(), err
}

// SoC implements the api.Vehicle interface
func (v *Cloud) SoC() (float64, error) {
	return v.chargeStateG()
}
