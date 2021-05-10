package server

import (
	"context"
	"log"

	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/soc/server/auth"
)

type AuthServer struct {
	pb.UnimplementedAuthServer
}

func (s *AuthServer) IsAuthorized(ctx context.Context, r *pb.AuthRequest) (*pb.AuthReply, error) {
	authorized, _, claims, err := isAuthorized(r)

	// track auth requests
	log.Printf(claims.Subject+": %v", authorized)

	res := &pb.AuthReply{Authorized: authorized}
	if err == nil {
		res.Subject = claims.Subject
	}

	return res, err
}

type tokenizer interface {
	GetToken() string
}

func isAuthorized(r tokenizer) (bool, string, *auth.Claims, error) {
	token := r.GetToken()

	claims, err := auth.ParseToken(token)
	if err != nil {
		return false, token, claims, err
	}

	authorized := claims.SponsorExempt
	if !authorized {
		authorized, err = auth.IsSponsor(claims.Subject)
	}

	return authorized, token, claims, err
}
