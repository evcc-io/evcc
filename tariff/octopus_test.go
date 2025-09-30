package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
)

func TestOctopusConfigParse(t *testing.T) {
	test.SkipCI(t)

	// This test will start failing if you remove the deprecated "tariff" config var.
	validTariffConfig := map[string]interface{}{
		"region":          "H",
		"tariff":          "GO-22-03-29",
		"tariffDirection": "import",
		"directDebit":     "True",
	}

	_, err := NewOctopusFromConfig(validTariffConfig)
	require.NoError(t, err)

	validProductCodeConfig := map[string]interface{}{
		"region":          "H",
		"productcode":     "GO-22-03-29",
		"tariffDirection": "import",
		"directDebit":     "False",
	}

	_, err = NewOctopusFromConfig(validProductCodeConfig)
	require.NoError(t, err)

	invalidApiAndProductCodeConfig := map[string]interface{}{
		"region":          "H",
		"productcode":     "GO-22-03-29",
		"tariffDirection": "import",
		"apikey":          "invalid_key",
	}
	_, err = NewOctopusFromConfig(invalidApiAndProductCodeConfig)
	require.Error(t, err)

	invalidTariffDirectionConfig := map[string]interface{}{
		"tariffDirection": "invalid",
		"apikey":          "test",
	}
	_, err = NewOctopusFromConfig(invalidTariffDirectionConfig)
	require.Errorf(t, err, "invalid tariff type")
}
