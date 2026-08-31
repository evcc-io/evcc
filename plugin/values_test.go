package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// intValuesFromYaml returns the values accepted by the yaml-defined int setter
func intValuesFromYaml(t *testing.T, s string) []int64 {
	t.Helper()

	var other map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(s), &other))

	var cfg Config
	require.NoError(t, util.DecodeOther(other, &cfg))

	return cfg.intValues(t.Context())
}

func TestSwitchIntValues(t *testing.T) {
	// cases without default
	require.Equal(t, []int64{1, 2}, intValuesFromYaml(t, `
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

	// an erroring default leaves the cases as the accepted values
	require.Equal(t, []int64{1}, intValuesFromYaml(t, `
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

	// a case that only errors out is not accepted, whether directly or nested
	require.Equal(t, []int64{1}, intValuesFromYaml(t, `
source: switch
switch:
- case: 1
  set:
    source: const
    value: a
- case: 2
  set:
    source: error
    error: ErrNotAvailable
- case: 3
  set:
    source: sequence
    set:
    - source: const
      value: a
    - source: error
      error: ErrNotAvailable
`))

	// a default that sets accepts any value
	require.Nil(t, intValuesFromYaml(t, `
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

func TestWrappedIntValues(t *testing.T) {
	// watchdog forwards its set config
	require.Equal(t, []int64{1, 2}, intValuesFromYaml(t, `
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

	// sequence accepts what all of its setters accept
	require.Equal(t, []int64{2}, intValuesFromYaml(t, `
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

	// a plugin that doesn't restrict its values
	require.Nil(t, intValuesFromYaml(t, `
source: const
value: a
`))
}
