package server

import (
	"context"
	"sync/atomic"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/internal/vehicle/cloud"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/soc/server/auth"
)

var vehicleID int64

type Server struct {
	vehicles map[string]map[int64]api.Vehicle
	pb.UnimplementedVehicleServer
}

type tokenizer interface {
	GetToken() string
}

func (s *Server) isAuthorized(r tokenizer) (string, error) {
	token := r.GetToken()

	user, err := auth.ParseToken(token)
	if err != nil {
		return token, err
	}

	authorized, err := auth.IsAuthorized(user)
	if err == nil && !authorized {
		err = cloud.ErrNotAuthorized
	}

	return token, err
}

type vehicler interface {
	tokenizer
	GetVehicleId() int64
}

func (s *Server) vehicle(r vehicler) (api.Vehicle, error) {
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

func (s *Server) New(ctx context.Context, r *pb.NewRequest) (*pb.NewReply, error) {
	token, err := s.isAuthorized(r)
	if err != nil {
		return nil, err
	}

	typ := r.GetType()
	config := r.GetConfig()

	v, err := vehicle.NewFromConfig(typ, stringMapToInterface(config))
	if err != nil {
		return nil, err
	}

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

func (s *Server) SoC(ctx context.Context, r *pb.SoCRequest) (*pb.SoCReply, error) {
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
