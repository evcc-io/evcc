package server

import (
	"testing"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	submitted := []ocpp.MirrorRule{{UpstreamURL: "wss://copy", Password: masked, CaCert: masked}}
	require.NoError(t, restoreMaskedMirrors("s", stored, submitted))
	assert.Equal(t, "secret", submitted[0].Password)
	assert.Equal(t, "pem", submitted[0].CaCert)

	submitted = []ocpp.MirrorRule{{UpstreamURL: "wss://other", Password: masked}}
	assert.Error(t, restoreMaskedMirrors("s", stored, submitted), "a changed upstream has no secrets to restore")

	submitted = []ocpp.MirrorRule{{UpstreamURL: "wss://other", Password: "typed"}}
	assert.NoError(t, restoreMaskedMirrors("s", stored, submitted), "entered secrets need no restoring")
}

func TestRestoreMaskedMirrorsAfterStationIdEdit(t *testing.T) {
	stored := []ocpp.MirrorRule{{UpstreamURL: "wss://copy", UpstreamStationID: "old", Password: "secret"}}

	submitted := []ocpp.MirrorRule{{UpstreamURL: "wss://copy/", UpstreamStationID: "new", Password: masked}}
	require.NoError(t, restoreMaskedMirrors("s", stored, submitted))
	assert.Equal(t, "secret", submitted[0].Password)

	stored = append(stored, ocpp.MirrorRule{UpstreamURL: "wss://copy", UpstreamStationID: "other", Password: "second"})
	submitted = []ocpp.MirrorRule{{UpstreamURL: "wss://copy", UpstreamStationID: "new", Password: masked}}
	assert.Error(t, restoreMaskedMirrors("s", stored, submitted), "ambiguous: two stored mirrors share the URL")
}
