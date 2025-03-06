package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO add JSON inline tags to Named/Typed structs
func testJson(t *testing.T) {
	c := Named{
		Name: "test",
		Type: "test",
		Other: map[string]any{
			"foo": "bar",
		},
	}

	b, err := json.Marshal(c)
	require.NoError(t, err)

	assert.Equal(t, `{"name":"test","type":"test","foo":"bar"}`, string(b))
}
