package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
)

func TestOctopusConfigParse(t *testing.T) {
	test.SkipCI(t)

	validTestApiKey32 := "oe_test_testingYqLeoRu2xsn9WEiv6"
	validTestApiKey40 := "oe_test_testingYBsFMxfqXG9guAdTVgFssdJmv"

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
		"apikey":          validTestApiKey32,
	}
	_, err = buildOctopusFromConfig(invalidTariffDirectionConfig)
	require.Errorf(t, err, "invalid tariff type")

	validApiExportConfig32 := map[string]any{
		"tariffDirection": "export",
		"apikey":          validTestApiKey32,
	}
	_, err = buildOctopusFromConfig(validApiExportConfig32)
	require.NoError(t, err)

	validApiExportConfig40 := map[string]any{
		"tariffDirection": "export",
		"apikey":          validTestApiKey40,
	}
	_, err = buildOctopusFromConfig(validApiExportConfig40)
	require.NoError(t, err)
}
