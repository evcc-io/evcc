package charger

import (
	"bytes"
	"fmt"
	"strings"
)

// ensureCharger extracts VIN from list of VINs returned from `list` function
func ensureCharger(vin string, list func() ([]string, error)) (string, error) {
	vin, _, err := ensureChargerWithFeature(vin, list, func(v string) (string, string) {
		return v, ""
	})

	return vin, err
}

// ensureChargerWithFeature extracts VIN and feature from list of chargers of type V returned from `list` function
func ensureChargerWithFeature[Charger, Feature any](
	vin string,
	list func() ([]Charger, error),
	extract func(Charger) (string, Feature),
) (string, Feature, error) {
	chargers, err := list()
	if err != nil {
		return "", *new(Feature), fmt.Errorf("cannot get chargers: %w", err)
	}

	if vin = strings.ToUpper(vin); vin != "" {
		for _, charger := range chargers {
			if v, res := extract(charger); strings.ToUpper(v) == vin {
				return v, res, nil
			}
		}

		// vin defined but doesn't exist
		err = fmt.Errorf("cannot find charger %s", vin)
	} else {
		// vin empty
		if len(chargers) == 1 {
			vin, res := extract(chargers[0])
			return vin, res, nil
		}

		err = fmt.Errorf("cannot find charger, got: %v", chargers)
	}

	return "", *new(Feature), err
}

// bytesAsString normalises a string by stripping leading 0x00 and trimming white space
func bytesAsString(b []byte) string {
	return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00")))
}
