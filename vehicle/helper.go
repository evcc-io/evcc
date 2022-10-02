package vehicle

import (
	"fmt"
	"strings"
)

// ensureVehicle extracts VIN from list of VINs returned from `list` function
func ensureVehicle(vin string, list func() ([]string, error)) (string, error) {
	return ensureVehicleEx(vin, list, func(v string) string {
		return v
	})
}

// ensureVehicleEx extracts vehicle with matching VIN from list of vehicles
func ensureVehicleEx[Vehicle any](
	vin string,
	list func() ([]Vehicle, error),
	extract func(Vehicle) string,
) (Vehicle, error) {
	vehicles, err := list()
	if err != nil {
		return *new(Vehicle), fmt.Errorf("cannot get vehicles: %w", err)
	}

	if vin = strings.ToUpper(vin); vin != "" {
		for _, vehicle := range vehicles {
			if vin == extract(vehicle) {
				return vehicle, nil
			}
		}

		// vin defined but doesn't exist
		err = fmt.Errorf("cannot find vehicle: %s", vin)
	} else {
		// vin empty
		if len(vehicles) == 1 {
			return vehicles[0], nil
		}

		err = fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	return *new(Vehicle), err
}
