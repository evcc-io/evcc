package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// intKeysFromYaml returns the keys the yaml-defined int setter switches on
func intKeysFromYaml(t *testing.T, s string) []int64 {
	t.Helper()

	var other map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(s), &other))

	var cfg Config
	require.NoError(t, util.DecodeOther(other, &cfg))

	return cfg.intKeys(t.Context())
}

func TestSwitchIntKeys(t *testing.T) {
	require.Equal(t, []int64{1, 2}, intKeysFromYaml(t, `
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

	// a default that only errors out leaves the cases as the keys
	require.Equal(t, []int64{1}, intKeysFromYaml(t, `
source: switch
switch:
- case: 1
  set:
    source: const
    value: a
default:
  source: error
  error: ErrNotAvailable
`))

	// a default that sets handles every other value, so there is no fixed key set
	require.Nil(t, intKeysFromYaml(t, `
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
	require.Equal(t, []int64{1, 2}, intKeysFromYaml(t, `
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

	// sequence keeps the keys all of its setters switch on
	require.Equal(t, []int64{2}, intKeysFromYaml(t, `
source: sequence
set:
- source: const
  value: a
- source: switch
  switch:
  - case: 1
    set:
      source: const
      value: a
  - case: 2
    set:
      source: const
      value: b
- source: switch
  switch:
  - case: 2
    set:
      source: const
      value: c
`))

	// a plugin that doesn't switch on its value
	require.Nil(t, intKeysFromYaml(t, `
source: const
value: a
`))
}
