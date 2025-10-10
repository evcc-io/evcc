//go:build integration
// +build integration

package subaru

import (
	"fmt"
	"os"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	// Skip if no credentials provided
	user := os.Getenv("SUBARU_USER")
	password := os.Getenv("SUBARU_PASSWORD")
	if user == "" || password == "" {
		t.Skip("SUBARU_USER or SUBARU_PASSWORD not set")
	}

	// Create and login identity
	util.LogLevel("trace", nil) // Enable trace logging
	log := util.NewLogger("test")
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
