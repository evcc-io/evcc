package octopuskraken

import (
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
)

func device(id, name, typ, provider string) Device {
	return Device{ID: id, Name: name, DeviceType: typ, Provider: provider}
}

func TestDeviceOptions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		devices []Device
		want    []service.Option
	}{
		{"empty", nil, []service.Option{}},
		{"sorted by id, vehicles match", []Device{
			device("dev-2", "Wallbox", "CHARGE_POINTS", "OCPP"),
			device("dev-1", "My Car", "ELECTRIC_VEHICLES", "TESLA"),
		}, []service.Option{
			{Value: "dev-1", Label: "My Car", Hint: "Vehicle · TESLA", Match: true},
			{Value: "dev-2", Label: "Wallbox", Hint: "Charge point · OCPP"},
		}},
		{"unknown type and missing provider", []Device{
			device("dev-3", "Heat Pump", "HEAT_PUMPS", ""),
		}, []service.Option{
			{Value: "dev-3", Label: "Heat Pump", Hint: "HEAT_PUMPS"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, deviceOptions(tc.devices))
		})
	}
}
