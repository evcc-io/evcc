package vehicle

import "github.com/evcc-io/evcc/util"

// ensureVehicle extracts VIN from list of VINs returned from `list` function
func ensureVehicle(vin string, list func() ([]string, error)) (string, error) {
	return ensureVehicleEx(vin, list, func(v string) (string, error) {
		return v, nil
	})
}

// ensureVehicleEx extracts vehicle with matching VIN from list of vehicles
func ensureVehicleEx[T any](
	vin string,
	list func() ([]T, error),
	extract func(T) (string, error),
) (T, error) {
	return util.EnsureElementEx(vin, list, extract)
}
