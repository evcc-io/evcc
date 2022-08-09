package vehicle

import (
	"fmt"
	"strings"
)

// ensureVehicleWithFeature extracts VIN from list of VINs returned from `list` function
func ensureVehicle(vin string, list func() ([]string, error)) (string, error) {
	vin, _, err := ensureVehicleWithFeature(vin, list, func(v string) (string, string) {
		return v, ""
	})

	return vin, err
}

// ensureVehicleWithFeature extracts VIN and feature from list of vehicles of type V returned from `list` function
func ensureVehicleWithFeature[Vehicle, Feature any](
	vin string,
	list func() ([]Vehicle, error),
	extract func(Vehicle) (string, Feature),
) (string, Feature, error) {
	vehicles, err := list()
	if err != nil {
		return "", *new(Feature), fmt.Errorf("cannot get vehicles: %w", err)
	}

	if vin = strings.ToUpper(vin); vin != "" {
		for _, vehicle := range vehicles {
			if v, res := extract(vehicle); v == vin {
				return v, res, nil
			}
		}

		// vin defined but doesn't exist
		err = fmt.Errorf("cannot find vehicle: %s", vin)
	} else {
		// vin empty
		if len(vehicles) == 1 {
			vin, res := extract(vehicles[0])
			return vin, res, nil
		}

		err = fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	return "", *new(Feature), err
}
