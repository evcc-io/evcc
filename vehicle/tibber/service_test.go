package tibber

import (
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
)

func TestVehicleOptions(t *testing.T) {
	vehicles := []Device{
		{ExternalID: "tesla:5YJSA1E26MF1234567", Info: DeviceInfo{Name: "Model S", Brand: "Tesla", Model: "Model S"}},
		{ExternalID: "vw:WVWZZZ1KZAW000001", Info: DeviceInfo{Name: " ", Brand: "Volkswagen", Model: "ID.3"}},
		{ExternalID: "skoda:TMBJJ7NX0L0000001"},
	}

	assert.Equal(t, []service.Option{
		{Value: "5YJSA1E26MF1234567", Label: "Model S", Hint: "Tesla Model S", Match: true},
		{Value: "TMBJJ7NX0L0000001", Match: true},
		{Value: "WVWZZZ1KZAW000001", Label: "Volkswagen ID.3", Match: true},
	}, vehicleOptions(vehicles))

	assert.Empty(t, vehicleOptions(nil))
}
