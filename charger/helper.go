package charger

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

// ensureCharger extracts ID from list of IDs returned from `list` function
func ensureCharger(id string, list func() ([]string, error)) (string, error) {
	id, _, err := ensureChargerWithFeature(id, list, func(v string) (string, string) {
		return v, ""
	})

	return id, err
}

// ensureChargerWithFeature extracts ID and feature from list of chargers of type V returned from `list` function
func ensureChargerWithFeature[Charger, Feature any](
	id string,
	list func() ([]Charger, error),
	extract func(Charger) (string, Feature),
) (string, Feature, error) {
	var zero Feature

	chargers, err := list()
	if err != nil {
		return "", zero, fmt.Errorf("cannot get chargers: %w", err)
	}

	if id = strings.ToUpper(id); id != "" {
		for _, charger := range chargers {
			if v, res := extract(charger); strings.ToUpper(v) == id {
				return v, res, nil
			}
		}

		// id defined but doesn't exist
		err = fmt.Errorf("cannot find charger %s", id)
	} else {
		// id empty
		if len(chargers) == 1 {
			id, res := extract(chargers[0])
			return id, res, nil
		}

		err = fmt.Errorf("cannot find charger, got: %v", chargers)
	}

	return "", zero, err
}

// bytesAsString normalises a string by stripping leading 0x00 and trimming white space
func bytesAsString(b []byte) string {
	return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00")))
}

// verifyEnabled validates the enabled state against the charger status
func verifyEnabled(c api.Charger, enabled bool) (bool, error) {
	if enabled {
		return true, nil
	}

	status, err := c.Status()

	// always treat charging as enabled
	return status == api.StatusC, err
}
