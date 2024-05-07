package tariff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOctopusConfigParse(t *testing.T) {
	// This test will start failing if you remove the deprecated "tariff" config var.
	validTariffConfig := map[string]interface{}{
		"region": "H",
		"tariff": "GO-22-03-29",
	}

	_, err := NewOctopusFromConfig(validTariffConfig)
	require.NoError(t, err)

	validProductCodeConfig := map[string]interface{}{
		"region":      "H",
		"productcode": "GO-22-03-29",
	}

	_, err = NewOctopusFromConfig(validProductCodeConfig)
	require.NoError(t, err)

	invalidApiAndProductCodeConfig := map[string]interface{}{
		"region":      "H",
		"productcode": "GO-22-03-29",
		"apikey":      "nope",
	}
	_, err = NewOctopusFromConfig(invalidApiAndProductCodeConfig)
	require.Error(t, err)
}
