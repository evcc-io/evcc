package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigHash(t *testing.T) {
	conf := &globalconfig.All{
		Meters: []config.Named{
			{Name: "m1", Type: "template", Other: map[string]any{"template": "sma"}},
			{Name: "m2", Type: "custom"},
		},
		Chargers:   []config.Named{{Name: "c1", Type: "template", Other: map[string]any{"template": "keba"}}},
		Loadpoints: []config.Named{{Name: "lp1"}},
		Tariffs:    globalconfig.Tariffs{Grid: config.Typed{Type: "template", Other: map[string]any{"template": "tibber"}}},
	}

	hash := configHash(conf)
	assert.NotEmpty(t, hash)
	assert.Equal(t, hash, configHash(conf), "must be deterministic")

	// order of devices must not change the hash
	reordered := *conf
	reordered.Meters = []config.Named{conf.Meters[1], conf.Meters[0]}
	assert.Equal(t, hash, configHash(&reordered), "must be order-independent")

	// an additional device must change the hash
	added := *conf
	added.Chargers = append(append([]config.Named{}, conf.Chargers...), config.Named{Name: "c2", Type: "easee"})
	assert.NotEqual(t, hash, configHash(&added), "different config must differ")
}
