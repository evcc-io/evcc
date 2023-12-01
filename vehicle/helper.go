package vehicle

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// ensureVehicle extracts VIN from list of VINs returned from `list` function
func ensureVehicle(vin string, list func() ([]string, error)) (string, error) {
	return ensureVehicleEx(vin, list, func(v string) string {
		return v
	})
}

// ensureVehicleEx extracts vehicle with matching VIN from list of vehicles
func ensureVehicleEx[T any](
	vin string,
	list func() ([]T, error),
	extract func(T) string,
) (T, error) {
	var zero T

	vehicles, err := list()
	if err != nil {
		return zero, fmt.Errorf("cannot get vehicles: %w", err)
	}

	// vin defined
	if vin = strings.ToUpper(vin); vin != "" {
		for _, vehicle := range vehicles {
			if vin == extract(vehicle) {
				return vehicle, nil
			}
		}

		// vin defined but doesn't exist
		return zero, fmt.Errorf("cannot find vehicle: %s", vin)
	}

	// vin empty
	if len(vehicles) == 1 {
		return vehicles[0], nil
	}

	return zero, fmt.Errorf("cannot find vehicle, got: %v", lo.Map(vehicles, func(v T, _ int) string {
		return extract(v)
	}))
}
