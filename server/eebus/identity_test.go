package eebus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want identity
	}{
		{
			name: "defaults",
			want: identity{vendor: "EVCC", brand: "EVCC", model: "HEMS", serial: "EVCC-1234"},
		},
		{
			name: "brand, model and serial",
			env:  map[string]string{"EEBUS_BRAND": "Acme", "EEBUS_MODEL": "Energy Manager", "EEBUS_SERIAL": "AC0001"},
			want: identity{vendor: "Acme", brand: "Acme", model: "Energy Manager", serial: "AC0001"},
		},
		{
			name: "vendor",
			env:  map[string]string{"EEBUS_BRAND": "Acme", "EEBUS_VENDOR": "Acme Inc."},
			want: identity{vendor: "Acme Inc.", brand: "Acme", model: "HEMS", serial: "EVCC-1234"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"EEBUS_BRAND", "EEBUS_MODEL", "EEBUS_VENDOR", "EEBUS_SERIAL"} {
				t.Setenv(key, tc.env[key])
			}
			assert.Equal(t, tc.want, deviceIdentity("EVCC-1234"))
		})
	}
}
