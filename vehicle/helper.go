package vehicle

import (
	"fmt"
	"strings"

	"github.com/thoas/go-funk"
)

// findVehicle finds the first vehicle in the list of VINs or returns an error
func findVehicle(vehicles []string, err error) (string, error) {
	if err != nil {
		return "", fmt.Errorf("cannot get vehicles: %w", err)
	}

	if len(vehicles) != 1 {
		return "", fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	vin := strings.TrimSpace(vehicles[0])
	if vin == "" {
		return "", fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	return vin, nil
}

// ensureVehicle ensures that the vehicle is available on the api and returns the VIN
func ensureVehicle(vin string, fun func() ([]string, error)) (string, error) {
	vehicles, err := fun()
	if err != nil {
		return "", fmt.Errorf("cannot get vehicles: %w", err)
	}

	if vin = strings.ToUpper(vin); vin != "" {
		// vin defined but doesn't exist
		if !funk.ContainsString(vehicles, vin) {
			err = fmt.Errorf("cannot find vehicle: %s", vin)
		}
	} else {
		// vin empty
		vin, err = findVehicle(vehicles, nil)
	}

	return vin, err
}
