package server

import (
	"context"

	"github.com/andig/evcc/internal/vehicle/cloud"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/soc/server/auth"
)

type AuthServer struct {
	pb.UnimplementedAuthServer
}

func (s *AuthServer) IsAuthorized(ctx context.Context, r *pb.AuthRequest) (*pb.AuthReply, error) {
	_, _, err := isAuthorized(r)
	return &pb.AuthReply{}, err
}

type tokenizer interface {
	GetToken() string
}

func isAuthorized(r tokenizer) (string, *auth.Claims, error) {
	token := r.GetToken()

	claims, err := auth.ParseToken(token)
	if err != nil {
		return token, claims, err
	}

	authorized, err := auth.IsAuthorized(claims.Subject)
	if err == nil && !authorized {
		err = cloud.ErrNotAuthorized
	}

	return token, claims, err
}
