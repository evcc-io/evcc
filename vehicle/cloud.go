package vehicle

import (
	"context"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/cloud"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Cloud is an api.Vehicle implementation
type Cloud struct {
	*embed
	token        string
	brand        string
	config       map[string]string
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
		embed `mapstructure:",squash"`
		Brand string
		Other map[string]string `mapstructure:",remain"`
		Cache time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	host := util.Getenv("GRPC_URI", cloud.Host)
	conn, err := cloud.Connection(host)
	if err != nil {
		return nil, err
	}

	v := &Cloud{
		embed:  &cc.embed,
		token:  sponsor.Token,
		brand:  cc.Brand,
		config: cc.Other,
		client: pb.NewVehicleClient(conn),
	}

	if err == nil {
		err = v.prepareVehicle()
	}

	v.chargeStateG = provider.Cached(v.chargeState, cc.Cache)

	return v, err
}

// prepareVehicle obtains new vehicle handle from cloud server
func (v *Cloud) prepareVehicle() error {
	req := &pb.NewRequest{
		Token:  v.token,
		Type:   v.brand,
		Config: v.config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*request.Timeout)
	defer cancel()

	res, err := v.client.New(ctx, req)
	if err == nil {
		v.vehicleID = res.VehicleId
	}

	return err
}

// chargeState implements the api.Vehicle interface
func (v *Cloud) chargeState() (float64, error) {
	req := &pb.SocRequest{
		Token:     v.token,
		VehicleId: v.vehicleID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	res, err := v.client.Soc(ctx, req)

	if err != nil && strings.Contains(err.Error(), api.ErrMustRetry.Error()) {
		return 0, api.ErrMustRetry
	}

	if err != nil && strings.Contains(err.Error(), cloud.ErrVehicleNotAvailable.Error()) && v.prepareVehicle() == nil {
		req.VehicleId = v.vehicleID
		res, err = v.client.Soc(ctx, req)
	}

	return res.GetSoc(), err
}

// Soc implements the api.Vehicle interface
func (v *Cloud) Soc() (float64, error) {
	return v.chargeStateG()
}
