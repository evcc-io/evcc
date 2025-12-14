package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeCustomSend(t *testing.T) {
	services := []config.Typed{
		{
			Type: "custom",
			Other: map[string]any{
				"encoding": "title",
				"send": map[string]string{
					"source": "script",
					"cmd":    "/usr/local/bin/evcc_message \"{{.send}}\"",
				},
			},
		},
	}
	expectedServices := []config.Typed{
		{
			Type: "custom",
			Other: map[string]any{
				"encoding": "title",
				"send":     "cmd: /usr/local/bin/evcc_message \"{{.send}}\"\nsource: script",
			},
		},
	}

	if err := normalizeMessagingCustomSend(services); err != nil {
		t.Fatalf("normalizeCustomSend returned error: %v", err)
	}

	assert.Equal(t, expectedServices, services)
}
