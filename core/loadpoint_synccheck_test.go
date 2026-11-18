package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func loadpointWithFeatures(features []api.Feature) *Loadpoint {
	return &Loadpoint{
		charger: struct {
			api.Charger
			api.FeatureDescriber
		}{
			FeatureDescriber: &featureCharger{features: features},
		},
	}
}

// TestEnableStateIsAdvisory: only a device that runs on its own schedule may answer "disabled" while
// the loadpoint asked for enabled - for everything else the mismatch is a fault worth a warning
func TestEnableStateIsAdvisory(t *testing.T) {
	for _, tc := range []struct {
		name     string
		features []api.Feature
		advisory bool
	}{
		{"heat pump", []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice}, true},
		{"switched heat pump", []api.Feature{api.Heating, api.IntegratedDevice, api.SwitchDevice}, false},
		{"wallbox", []api.Feature{api.SwitchDevice}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.advisory, loadpointWithFeatures(tc.features).enableStateIsAdvisory())
		})
	}
}
