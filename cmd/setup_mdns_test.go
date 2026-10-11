package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMdnsInstance(t *testing.T) {
	id := strings.Repeat("a", 32)

	name := mdnsInstance(id)
	assert.Regexp(t, `^evcc-[0-9a-f]{4}$`, name)
	assert.Equal(t, name, mdnsInstance(id), "stable")
	assert.NotEqual(t, name, mdnsInstance(strings.Repeat("b", 32)), "unique")
}
