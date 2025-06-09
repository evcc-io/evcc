package charger

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
	"golang.org/x/text/encoding/unicode"
)

// TODO remove when used
var _ = ensureCharger

// ensureCharger extracts ID from list of IDs returned from `list` function
func ensureCharger(id string, list func() ([]string, error)) (string, error) {
	id, err := ensureChargerEx(id, list, func(v string) (string, error) {
		return v, nil
	})

	return id, err
}

// ensureChargerEx extracts charger with matching id from list of chargers
func ensureChargerEx[T any](
	id string,
	list func() ([]T, error),
	extract func(T) (string, error),
) (T, error) {
	var zero T

	chargers, err := list()
	if err != nil {
		return zero, fmt.Errorf("cannot get chargers: %w", err)
	}

	if id = strings.ToUpper(id); id != "" {
		for _, charger := range chargers {
			cc, err := extract(charger)
			if err != nil {
				return zero, err
			}
			if strings.ToUpper(cc) == id {
				return charger, nil
			}
		}
	} else if len(chargers) == 1 {
		// id empty and exactly one charger
		return chargers[0], nil
	}

	return zero, fmt.Errorf("cannot find charger, got: %v", lo.Map(chargers, func(v T, _ int) string {
		vin, _ := extract(v)
		return vin
	}))
}

// bytesAsString normalises a string by stripping leading 0x00 and trimming white space
func bytesAsString(b []byte) string {
	return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00")))
}

// utf16BEBytesAsString converts a byte slice containing UTF-16 Big-Endian encoded text to a string and trims white spaces
func utf16BEBytesAsString(b []byte) (string, error) {
	s, err := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder().String(string(bytes.TrimRight(b, "\x00")))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
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
