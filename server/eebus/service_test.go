package eebus

import (
	"testing"

	shipapi "github.com/enbility/ship-go/api"
	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
)

func TestDeviceOptions(t *testing.T) {
	services := []shipapi.RemoteMdnsService{
		{Ski: "bbb", Name: "Wallbox", ShipID: "Wallbox-2", Brand: "Elli", Model: "Charger Pro", Serial: "12345"},
		{Ski: "aaa", Name: "Steuerbox", ShipID: "FNN-1"},
		{Ski: "ccc", ShipID: "Heatpump-3", Brand: "Vaillant"},
	}
	used := func(ski string) bool { return ski == "bbb" }

	assert.Equal(t, []service.Option{
		{Value: "aaa", Label: "Steuerbox", Hint: "FNN-1", Match: true},
		{Value: "bbb", Label: "Elli Charger Pro", Hint: "12345", Match: true, Used: true},
		{Value: "ccc", Label: "Vaillant", Hint: "Heatpump-3", Match: true},
	}, deviceOptions(services, used))

	assert.Equal(t, []service.Option{}, deviceOptions(nil, used))
}
