package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryMatch(t *testing.T) {
	d := Discovery{
		Mdns:     []string{"_shelly._tcp", "_http._tcp:shelly*"},
		Hostname: []string{"sma*"},
		Mac:      []string{"0015BB", "C0619AB"},
	}

	tc := []struct {
		name      string
		hostnames []string
		mac       string
		services  []string
		expected  bool
	}{
		{"mac", nil, "00:15:bb:12:34:56", nil, true},
		{"mac 28 bit", nil, "C0:61:9A:B1:23:45", nil, true},
		{"mac 28 bit other vendor", nil, "C0:61:9A:C1:23:45", nil, false},
		{"hostname", []string{"SMA3009876543.fritz.box"}, "", nil, true},
		{"hostname alias", []string{"inverter.fritz.box", "sma3009876543.local"}, "", nil, true},
		{"hostname other", []string{"plasma"}, "", nil, false},
		{"mdns type", nil, "", []string{"_shelly._tcp:any"}, true},
		{"mdns instance", nil, "", []string{"_http._tcp:ShellyPlug-A8032AB"}, true},
		{"mdns instance other", nil, "", []string{"_http._tcp:printer"}, false},
		{"empty", nil, "", nil, false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, d.Match(tc.hostnames, tc.mac, tc.services))
		})
	}

	assert.False(t, Discovery{}.Match([]string{"sma1"}, "00:15:BB:12:34:56", []string{"_shelly._tcp:any"}))
	assert.Equal(t, []string{"_shelly._tcp", "_http._tcp"}, d.MdnsTypes())
}

func TestDiscoveryValidate(t *testing.T) {
	require.NoError(t, Discovery{
		Mdns: []string{"_shelly._tcp", "_http._tcp:shelly*", "_go-e_go-eCharger._tcp"},
		Mac:  []string{"0015BB", "C0619AB", "70B3D5C45"},
	}.validate())

	for _, d := range []Discovery{
		{Mdns: []string{"shelly"}},
		{Mac: []string{"00:15:BB"}},
		{Mac: []string{"0015bb"}},
		{Hostname: []string{"sma["}},
	} {
		assert.Error(t, d.validate())
	}
}
