package ecoflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAppKeys(t *testing.T) {
	pair := func(i int) string {
		return fmt.Sprintf(`["%032x","%032x"]`, i, i+100)
	}

	var pairs []string
	for i := range 8 {
		pairs = append(pairs, pair(i+1))
	}
	bundle := `x=` + pair(99) + `,w=[` + strings.Join(pairs, ",") + `],y=1`

	keys, err := parseAppKeys([]byte(bundle))
	require.NoError(t, err)
	assert.Len(t, keys, 8)
	assert.Equal(t, AppKey{Key: fmt.Sprintf("%032x", 1), Secret: fmt.Sprintf("%032x", 101)}, keys[0])

	_, err = parseAppKeys([]byte(pair(1)))
	assert.Error(t, err, "a single pair is no key table")
}
