package eebus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeInterfaces(t *testing.T) {
	assert.Nil(t, normalizeInterfaces(nil))
	assert.Equal(t, []string{"eth0"}, normalizeInterfaces([]string{"eth0"}))
	assert.Equal(t, []string{"eth0", "eth1"}, normalizeInterfaces([]string{"eth0", "eth1"}))
	// comma-separated single string (env var or YAML scalar)
	assert.Equal(t, []string{"eth0", "eth1"}, normalizeInterfaces([]string{"eth0, eth1"}))
	// mixed + empty entries dropped
	assert.Equal(t, []string{"eth0", "eth1", "eth2"}, normalizeInterfaces([]string{" eth0 ,eth1", "", "eth2"}))
}
