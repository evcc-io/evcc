//go:build integration
// +build integration

package toyota

import (
	"fmt"
	"os"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	// Skip if no credentials provided
	user := os.Getenv("TOYOTA_USER")
	password := os.Getenv("TOYOTA_PASSWORD")
	if user == "" || password == "" {
		t.Skip("TOYOTA_USER or TOYOTA_PASSWORD not set")
	}

	// Create and login identity
	log := util.NewLogger("foo")
	identity := NewIdentity(log)
	err := identity.Login(user, password)
	require.NoError(t, err)

	// Create API client
	api := NewAPI(log, identity)

	// Test Vehicles method
	vehicles, err := api.Vehicles()
	require.NoError(t, err)
	fmt.Printf("Vehicles: %+v\n", vehicles)
	require.NotEmpty(t, vehicles, "expected at least one vehicle")

	for _, vin := range vehicles {
		require.NotEmpty(t, vin, "expected non-empty VIN")
	}

	// Test Status method for first vehicle
	status, err := api.Status(vehicles[0])
	require.NoError(t, err)
	require.NotNil(t, status)
	fmt.Printf("Vehicle status: %+v\n", status)
}
