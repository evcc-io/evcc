package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/polestar/legacy"
)

// PolestarGRPC is an api.Vehicle implementation for Polestar cars using the
// gRPC battery API. It restores the charging status that Polestar removed from
// the GraphQL API, see https://github.com/evcc-io/evcc/issues/30071.
type PolestarGRPC struct {
	*embed
	*legacy.GrpcProvider
}

func init() {
	registry.Add("polestar-grpc", NewPolestarGRPCFromConfig)
}

// NewPolestarGRPCFromConfig creates a new vehicle
func NewPolestarGRPCFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Cache          time.Duration
		Timeout        time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("polestar").Redact(cc.User, cc.Password, cc.VIN)

	v := &PolestarGRPC{
		embed: &cc.embed,
	}

	identity, err := legacy.NewIdentity(log, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := legacy.NewAPI(log, identity)

	vehicle, err := ensureVehicleEx(cc.VIN, func() ([]legacy.ConsumerCar, error) {
		ctx, cancel := context.WithTimeout(context.Background(), cc.Timeout)
		defer cancel()
		return api.Vehicles(ctx)
	}, func(v legacy.ConsumerCar) (string, error) {
		return v.VIN, nil
	})
	if err != nil {
		return v, err
	}

	grpcAPI, err := legacy.NewGrpcAPI(log, identity)
	if err != nil {
		return v, fmt.Errorf("grpc: %w", err)
	}

	v.GrpcProvider = legacy.NewGrpcProvider(grpcAPI, api, vehicle.VIN, cc.Timeout, cc.Cache)

	return v, nil
}
