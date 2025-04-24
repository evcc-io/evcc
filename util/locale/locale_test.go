package locale

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocales(t *testing.T) {
	require.NoError(t, Init())
	assert.Equal(t, "a", Bundle.LanguageTags())
}
