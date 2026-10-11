package smarthome

import (
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
)

func TestDeviceOptions(t *testing.T) {
	devices := []Device{
		{AIN: "11657 0123456", Name: "Wallbox", ProductName: " FRITZ!DECT 210 ", IsConnected: true},
		{AIN: "08761 0000001", Name: "Garage", ProductName: "", IsConnected: false},
	}

	assert.Equal(t, []service.Option{
		{Value: "08761 0000001", Label: "Garage"},
		{Value: "11657 0123456", Label: "Wallbox", Hint: "FRITZ!DECT 210", Match: true},
	}, deviceOptions(devices))

	assert.Empty(t, deviceOptions(nil))
}
