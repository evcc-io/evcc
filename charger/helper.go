package charger

import "github.com/evcc-io/evcc/api"

// switchStatus calculates a generic switches status
func switchStatus(
	enabled func() (bool, error),
	currentPower func() (float64, error),
	standbypower float64,
) (api.ChargeStatus, error) {
	res := api.StatusB

	// static mode
	if standbypower < 0 {
		on, err := enabled()
		if on {
			res = api.StatusC
		}

		return res, err
	}

	// standby power mode
	power, err := currentPower()
	if power > standbypower {
		res = api.StatusC
	}

	return res, err
}
