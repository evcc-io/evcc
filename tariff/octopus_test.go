package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
)

func TestOctopusConfigParse(t *testing.T) {
	test.SkipCI(t)

	// This test will start failing if you remove the deprecated "tariff" config var.
	validTariffConfig := map[string]any{
		"region":      "H",
		"tariff":      "GO-22-03-29",
		"directDebit": "True",
	}

	_, err := buildOctopusFromConfig(validTariffConfig)
	require.NoError(t, err)

	validProductCodeConfig := map[string]any{
		"region":      "H",
		"productcode": "GO-22-03-29",
		"directDebit": "False",
	}

	_, err = buildOctopusFromConfig(validProductCodeConfig)
	require.NoError(t, err)

	invalidApiAndProductCodeConfig := map[string]any{
		"region":          "H",
		"productcode":     "GO-22-03-29",
		"tariffDirection": "import",
		"apikey":          "invalid_key",
	}
	_, err = buildOctopusFromConfig(invalidApiAndProductCodeConfig)
	require.Error(t, err)

	invalidTariffDirectionConfig := map[string]any{
		"tariffDirection": "invalid",
		"apikey":          "test",
	}
	_, err = buildOctopusFromConfig(invalidTariffDirectionConfig)
	require.Errorf(t, err, "invalid tariff type")

	validApiExportConfig := map[string]any{
		"tariffDirection": "export",
		"apikey":          "test",
	}
	_, err = buildOctopusFromConfig(validApiExportConfig)
	require.NoError(t, err)
}
