package service

import (
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/discovery"
	"github.com/stretchr/testify/assert"
)

func TestMaskHostname(t *testing.T) {
	for in, expected := range map[string]string{
		"SUNGROWB23417A5199.local":         "SUNGROWB*",
		"shellyplugmg3-70af09e1482c.local": "shellyplugmg*",
		"go-echarger_237557.fritz.box":     "go-echarger_*",
		"fronius-symo-gen24":               "fronius-symo-gen*",
		"sma3009876543":                    "sma*",
		"1118431-abc":                      "*",
		"venus.local":                      "venus",
		"":                                 "",
	} {
		assert.Equal(t, expected, maskHostname(in), in)
	}
}

func TestMaskService(t *testing.T) {
	assert.Equal(t, "_e3dc._tcp:device-*", maskService("_e3dc._tcp:device-4711"))
	assert.Equal(t, "urn:schemas-upnp-org:device:fritzbox:1", maskService("urn:schemas-upnp-org:device:fritzbox:1"))
	assert.Empty(t, maskService("uuid:75802409-bccb-40e7-8e6c-3810770a4b56"))
}

func TestReportDevices(t *testing.T) {
	hosts := []discovery.Host{
		{IP: "192.0.2.1", MAC: "AA:BB:CC:00:00:01", Hostname: "phone-of-someone"},
		{
			IP: "192.0.2.158", MAC: "AC:19:9F:12:34:56", Hostname: "inverter.fritz.box",
			Aliases:  []string{"SUNGROWB23417A5199.fritz.box", "SUNGROWB23417A5199.local"},
			Services: []string{"_http._tcp:SUNGROWB23417A5199"},
		},
		{IP: "192.0.2.9", Hostname: "other.local"},
	}

	configs := []config.Named{
		{Type: "template", Other: map[string]any{"template": "sungrow-hybrid", "usage": "pv", "host": "192.0.2.158"}},
		{Type: "template", Other: map[string]any{"template": "sungrow-hybrid", "usage": "battery", "host": "192.0.2.158"}},
		{Type: "template", Other: map[string]any{"template": "uri", "uri": "http://OTHER.local:8080/api"}},
		{Type: "template", Other: map[string]any{"template": "cloud", "user": "someone"}},
		{Type: "template", Other: map[string]any{"template": "other", "host": "192.0.2.77"}},
		{Type: "custom", Other: map[string]any{"host": "192.0.2.1"}},
	}

	assert.Equal(t, []ReportDevice{
		{
			Template:  "sungrow-hybrid",
			Mac:       "AC199F",
			Hostnames: []string{"inverter", "SUNGROWB*"},
			Services:  []string{"_http._tcp:SUNGROWB*"},
		},
		{Template: "other"},
	}, reportDevices(configs, hosts))
}
