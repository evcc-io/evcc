package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// intKeysFromYaml returns the keys the yaml-defined int setter switches on
func intKeysFromYaml(t *testing.T, s string) ([]int64, error) {
	t.Helper()

	var other map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(s), &other))

	var cfg Config
	require.NoError(t, util.DecodeOther(other, &cfg))

	return cfg.intKeys(t.Context())
}

// mustIntKeys fails the test if the setter has no fixed set of keys
func mustIntKeys(t *testing.T, s string) []int64 {
	t.Helper()

	keys, err := intKeysFromYaml(t, s)
	require.NoError(t, err)

	return keys
}

func TestSwitchIntKeys(t *testing.T) {
	require.Equal(t, []int64{1, 2}, mustIntKeys(t, `
source: switch
switch:
- case: 1
  set:
    source: const
    value: a
- case: 2
  set:
    source: const
    value: b
`))

	// a default accepts the values without a case, so there is no fixed set of keys
	require.Nil(t, mustIntKeys(t, `
source: switch
switch:
- case: 1
  set:
    source: const
    value: a
default:
  source: const
  value: b
`))
}

func TestWrappedIntKeys(t *testing.T) {
	// watchdog forwards its set config
	require.Equal(t, []int64{1, 2}, mustIntKeys(t, `
source: watchdog
timeout: 1m
set:
  source: switch
  switch:
  - case: 1
    set:
      source: const
      value: a
  - case: 2
    set:
      source: const
      value: b
`))

	// a plugin that doesn't switch on its value
	require.Nil(t, mustIntKeys(t, `
source: const
value: a
`))

	// a sequence hides the switch, so its keys are not available
	require.Nil(t, mustIntKeys(t, `
source: sequence
set:
- source: switch
  switch:
  - case: 1
    set:
      source: const
      value: a
`))
}
