package server

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceConfigMapCustomYamlRedacted(t *testing.T) {
	const yamlStr = "type: template\ntemplate: foo\nmaxPower: 2760\npassword: secret123\n"

	dev := config.NewConfigurableDevice[api.Charger](&config.Config{
		ID:         1,
		Class:      templates.Charger,
		Properties: config.Properties{Type: "custom"},
		Data:       map[string]any{"yaml": yamlStr, "password": "secret123"},
	}, nil)

	for hidePrivate, want := range map[bool]string{false: "secret123", true: "*****"} {
		dc, err := deviceConfigMap(templates.Charger, dev, hidePrivate)
		require.NoError(t, err)

		conf := dc["config"].(map[string]any)
		assert.Contains(t, conf["yaml"], "password: "+want)
		assert.Contains(t, conf["yaml"], "maxPower: 2760")
		assert.Equal(t, want, conf["password"])
	}
}
