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

	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://main", UpstreamStationID: "s"}}
	assert.Error(t, validateMirrors(rule), "the charger's own id is the main upstream's effective id")

	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://copy"}, {UpstreamURL: "wss://copy", UpstreamStationID: "s"}}
	assert.Error(t, validateMirrors(rule), "listed twice under its effective id")

	rule.UpstreamStationID = "billing"
	rule.Mirrors = []ocpp.MirrorRule{{UpstreamURL: "wss://main", UpstreamStationID: "s"}}
	assert.NoError(t, validateMirrors(rule), "another station at the same server")
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

func TestRestoreMaskedMirrorsAfterStationIdEdit(t *testing.T) {
	stored := []ocpp.MirrorRule{{UpstreamURL: "wss://copy", UpstreamStationID: "old", Password: "secret"}}

	submitted := []ocpp.MirrorRule{{UpstreamURL: "wss://copy/", UpstreamStationID: "new", Password: masked}}
	restoreMaskedMirrors(stored, submitted)
	assert.Equal(t, "secret", submitted[0].Password)

	stored = append(stored, ocpp.MirrorRule{UpstreamURL: "wss://copy", UpstreamStationID: "other", Password: "second"})
	submitted = []ocpp.MirrorRule{{UpstreamURL: "wss://copy", UpstreamStationID: "new", Password: masked}}
	restoreMaskedMirrors(stored, submitted)
	assert.Empty(t, submitted[0].Password, "ambiguous: two stored mirrors share the URL")
}
