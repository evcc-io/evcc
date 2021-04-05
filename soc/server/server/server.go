package server

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/soc/server/auth"
)

var ErrNotAuthorized = errors.New("not authorized")

var vehicleID int64

type Server struct {
	vehicles map[int64]api.Vehicle
	pb.UnimplementedVehicleServer
}

type tokenizer interface {
	GetToken() string
}

func (s *Server) isAuthorized(r tokenizer) error {
	token := r.GetToken()

	user, err := auth.ParseToken(token)
	if err != nil {
		return err
	}

	authorized, err := auth.IsAuthorized(user)
	if err == nil && !authorized {
		err = ErrNotAuthorized
	}

	return err
}

func stringMapToInterface(in map[string]string) map[string]interface{} {
	res := make(map[string]interface{})

	for k, v := range in {
		res[k] = v
	}

	return res
}

func (s *Server) New(ctx context.Context, r *pb.NewRequest) (*pb.NewReply, error) {
	if err := s.isAuthorized(r); err != nil {
		return nil, err
	}

	typ := r.GetType()
	config := r.GetConfig()

	v, err := vehicle.NewFromConfig(typ, stringMapToInterface(config))
	if err != nil {
		return nil, err
	}

	id := atomic.AddInt64(&vehicleID, 1)
	s.vehicles[id] = v

	res := pb.NewReply{
		VehicleId: id,
	}

	return &res, nil
}

func (s *Server) SoC(ctx context.Context, r *pb.SoCRequest) (*pb.SoCReply, error) {
	if err := s.isAuthorized(r); err != nil {
		return nil, err
	}

	id := r.GetVehicleId()
	v, ok := s.vehicles[id]
	if !ok {
		return nil, errors.New("vehicle does not exist")
	}

	soc, err := v.SoC()

	res := pb.SoCReply{
		Soc: soc,
	}

	return &res, err
}
