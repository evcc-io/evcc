package soc

import "fmt"

// Guard checks soc value for validity
func Guard(soc float64, err error) (float64, error) {
	switch {
	case err != nil:
		return soc, err

	case soc < 0:
		return 0, fmt.Errorf("invalid soc: %.1f", soc)

	case soc > 100:
		return 100, fmt.Errorf("invalid soc: %.1f", soc)

	default:
		return soc, nil
	}
}
