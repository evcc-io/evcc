package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"sync/atomic"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/util/cloud"
)

var vehicleID int64

type VehicleContainer struct {
	id      int64
	hash    []byte
	vehicle api.Vehicle
}

type VehicleServer struct {
	registry map[string][]*VehicleContainer
	pb.UnimplementedVehicleServer
}

type vehicler interface {
	tokenizer
	GetVehicleId() int64
}

func (s *VehicleServer) vehicle(r vehicler) (api.Vehicle, error) {
	token := r.GetToken()
	vehicles, ok := s.registry[token]
	if !ok {
		return nil, cloud.ErrVehicleNotAvailable
	}

	id := r.GetVehicleId()
	for _, c := range vehicles {
		if c.id == id {
			return c.vehicle, nil
		}
	}

	return nil, cloud.ErrVehicleNotAvailable
}

func stringMapToInterface(in map[string]string) map[string]interface{} {
	res := make(map[string]interface{})

	for k, v := range in {
		res[k] = v
	}

	return res
}

func (s *VehicleServer) addVehicleToRegistry(token, typ string, config map[string]string, v api.Vehicle) int64 {
	id := atomic.AddInt64(&vehicleID, 1)

	// sort config keys
	var keys []string
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// hash config
	h := sha256.New()
	_, _ = h.Write([]byte(typ))
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(config[k]))
	}
	hash := h.Sum(nil)

	// find vehicle by hash and update it
	for _, c := range s.registry[token] {
		if bytes.Equal(c.hash, hash) {
			c.vehicle = v
			c.id = id
			return id
		}
	}

	// register new vehicle
	c := VehicleContainer{
		id:      id,
		hash:    hash,
		vehicle: v,
	}
	s.registry[token] = append(s.registry[token], &c)

	h.Reset()
	_, _ = h.Write([]byte(token))
	thash := fmt.Sprintf("%x", h.Sum(nil))

	updateActiveVehiclesMetric(thash, typ, 1)

	return id
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

	id := s.addVehicleToRegistry(token, typ, config, v)

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
