package vehicle

import (
	"fmt"
	"strings"
)

func findVehicle(vehicles []string, err error) (string, error) {
	if err != nil {
		return "", fmt.Errorf("cannot get vehicles: %v", err)
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
