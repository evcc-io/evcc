package vehicle

import (
	"fmt"
)

func findVehicle(vehicles []string, err error) (string, error) {
	if err != nil {
		return "", fmt.Errorf("cannot get vehicles: %v", err)
	}

	if len(vehicles) != 1 {
		return "", fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	return vehicles[0], nil
}
