package eebus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceIdentity(t *testing.T) {
	assert.Equal(t, identity{vendor: "EVCC", brand: "EVCC", model: "HEMS", serial: "EVCC-1234"}, deviceIdentity("EVCC-1234"))

	t.Setenv("EEBUS_BRAND", "Acme")
	t.Setenv("EEBUS_MODEL", "Energy Manager")
	t.Setenv("EEBUS_SERIAL", "AC0001")
	assert.Equal(t, identity{vendor: "Acme", brand: "Acme", model: "Energy Manager", serial: "AC0001"}, deviceIdentity("EVCC-1234"))

	t.Setenv("EEBUS_VENDOR", "Acme Inc.")
	assert.Equal(t, "Acme Inc.", deviceIdentity("EVCC-1234").vendor)
}
