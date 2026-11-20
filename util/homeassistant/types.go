package homeassistant

import (
	"fmt"
	"strings"
)

type ServiceDomainResponse struct {
	Domain   string         `json:"domain"`
	Services map[string]any `json:"services"`
}

type StateResponse struct {
	EntityId   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		UnitOfMeasurement string `json:"unit_of_measurement"`
		DeviceClass       string `json:"device_class"`
		FriendlyName      string `json:"friendly_name"`
	} `json:"attributes"`
}

func (state StateResponse) scale() (float64, error) {
	if unit, ok := strings.CutSuffix(state.Attributes.UnitOfMeasurement, "W"); ok {
		switch unit {
		case "":
		case "k":
			return 1e3, nil
		default:
			return 0, fmt.Errorf("invalid unit '%s'", state.Attributes.UnitOfMeasurement)
		}
	} else if unit, ok := strings.CutSuffix(state.Attributes.UnitOfMeasurement, "Wh"); ok {
		switch unit {
		case "":
			return 1e-3, nil
		case "k":
		case "M":
			return 1e3, nil
		default:
			return 0, fmt.Errorf("invalid unit '%s'", state.Attributes.UnitOfMeasurement)
		}
	}
	return 1, nil
}

// toCelsius converts a temperature value with the given Home Assistant unit to °C
func toCelsius(value float64, unit string) (float64, error) {
	switch unit {
	case "", "°C":
		return value, nil
	case "°F":
		return (value - 32) * 5 / 9, nil
	case "K":
		return value - 273.15, nil
	default:
		return 0, fmt.Errorf("invalid temperature unit '%s'", unit)
	}
}
