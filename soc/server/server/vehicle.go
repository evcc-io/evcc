package server

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/util/cloud"
)

var vehicleID int64

type VehicleServer struct {
	vehicles map[string]map[int64]api.Vehicle
	pb.UnimplementedVehicleServer
}

type vehicler interface {
	tokenizer
	GetVehicleId() int64
}

func (s *VehicleServer) vehicle(r vehicler) (api.Vehicle, error) {
	token := r.GetToken()
	vehicles, ok := s.vehicles[token]
	if !ok {
		return nil, cloud.ErrVehicleNotAvailable
	}

	id := r.GetVehicleId()
	v, ok := vehicles[id]
	if !ok {
		return nil, cloud.ErrVehicleNotAvailable
	}

	return v, nil
}

func stringMapToInterface(in map[string]string) map[string]interface{} {
	res := make(map[string]interface{})

	for k, v := range in {
		res[k] = v
	}

	return res
}

func (s *VehicleServer) New(ctx context.Context, r *pb.NewRequest) (*pb.NewReply, error) {
	authorized, token, claims, err := isAuthorized(r)
	if err != nil {
		return nil, err
	}
	if !authorized {
		return nil, cloud.ErrNotAuthorized
	}

	typ := r.GetType()
	config := r.GetConfig()

	// track vehicle create
	log.Println(claims.Subject+":", typ)

	v, err := vehicle.NewFromConfig(typ, stringMapToInterface(config))
	if err != nil {
		return nil, err
	}

	updateActiveVehiclesMetric(typ, 1)

	id := atomic.AddInt64(&vehicleID, 1)
	if s.vehicles[token] == nil {
		s.vehicles[token] = make(map[int64]api.Vehicle)
	}
	s.vehicles[token][id] = v

	res := pb.NewReply{
		VehicleId: id,
	}

	return &res, nil
}

func (s *VehicleServer) SoC(ctx context.Context, r *pb.SoCRequest) (*pb.SoCReply, error) {
	v, err := s.vehicle(r)
	if err != nil {
		return nil, err
	}

	soc, err := v.SoC()
	res := pb.SoCReply{
		Soc: soc,
	}

	return &res, err
}
