package server

import (
	"testing"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/stretchr/testify/assert"
)

func TestValidateMirrors(t *testing.T) {
	rule := ocpp.ForwarderRule{StationID: "s", UpstreamURL: "wss://main"}

	assert.NoError(t, validateMirrors(rule))

	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://copy"}}
	assert.NoError(t, validateMirrors(rule))

	rule.Mirrors = []ocpp.MirrorRule{{}}
	assert.Error(t, validateMirrors(rule), "url required")

	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://copy"}, {UpstreamURL: "wss://copy/"}}
	assert.Error(t, validateMirrors(rule), "duplicate")

	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://main/"}}
	assert.Error(t, validateMirrors(rule), "same as main upstream")
}

func TestRestoreMaskedMirrors(t *testing.T) {
	stored := []ocpp.MirrorRule{{UpstreamURL: "wss://copy", Password: "secret", CaCert: "pem"}}

	submitted := []ocpp.MirrorRule{
		{UpstreamURL: "wss://copy", Password: masked, CaCert: masked},
		{UpstreamURL: "wss://other", Password: masked},
	}
	restoreMaskedMirrors(stored, submitted)

	assert.Equal(t, "secret", submitted[0].Password)
	assert.Equal(t, "pem", submitted[0].CaCert)
	assert.Empty(t, submitted[1].Password, "mask must never be persisted")
}
