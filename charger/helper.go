package charger

import "github.com/evcc-io/evcc/api"

// switchStatus calcualates a generic switches status
func switchStatus(enabled bool, power, standbypower float64) api.ChargeStatus {
	res := api.StatusB

	// static mode || standby power mode condition
	if enabled && (standbypower < 0 || power > standbypower) {
		res = api.StatusC
	}

	return res
}
