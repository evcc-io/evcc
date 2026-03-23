package locale

import (
	"os"
	"testing"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocales(t *testing.T) {
	assets.I18n = os.DirFS("../../i18n")
	require.NoError(t, Init())
	assert.Less(t, 1, len(Bundle.LanguageTags()))
}
