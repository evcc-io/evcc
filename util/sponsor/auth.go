package sponsor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/util/cloud"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	mu             sync.RWMutex
	Subject, Token string
	ExpiresAt      time.Time
)

const (
	unavailable = "sponsorship unavailable"
	victron     = "victron"
)

func IsAuthorized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(Subject) > 0
}

func IsAuthorizedForApi() bool {
	mu.RLock()
	defer mu.RUnlock()
	return IsAuthorized() && Subject != unavailable && Token != ""
}

// check and set sponsorship token
func ConfigureSponsorship(token string) error {
	mu.Lock()
	defer mu.Unlock()

	if token == "" {
		if sub := checkVictron(); sub != "" {
			Subject = sub
			return nil
		}

		var err error
		if token, err = readSerial(); token == "" || err != nil {
			return err
		}
	}

	Token = token

	conn, err := cloud.Connection()
	if err != nil {
		return err
	}

	client := pb.NewAuthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := client.IsAuthorized(ctx, &pb.AuthRequest{Token: token})
	if err == nil && res.Authorized {
		Subject = res.Subject
		ExpiresAt = res.ExpiresAt.AsTime()
	}

	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
			Subject = unavailable
			err = nil
		} else {
			err = fmt.Errorf("sponsortoken: %w", err)
		}
	}

	return err
}

// redactToken returns a redacted version of the token showing only start and end characters
func redactToken(token string) string {
	if len(token) <= 12 {
		return ""
	}
	return token[:6] + "......." + token[len(token)-6:]
}

type Status struct {
	Name        string    `json:"name"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	ExpiresSoon bool      `json:"expiresSoon,omitempty"`
	Token       string    `json:"token,omitempty"`
}

// GetStatus returns the sponsorship status
func GetStatus() Status {
	mu.RLock()
	defer mu.RUnlock()

	var expiresSoon bool
	if d := time.Until(ExpiresAt); d < 30*24*time.Hour && d > 0 {
		expiresSoon = true
	}

	return Status{
		Name:        Subject,
		ExpiresAt:   ExpiresAt,
		ExpiresSoon: expiresSoon,
		Token:       redactToken(Token),
	}
}
