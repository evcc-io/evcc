package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMdnsInstance(t *testing.T) {
	for _, tc := range []struct {
		taken    []string
		expected string
	}{
		{nil, "evcc"},
		{[]string{"printer", "evcc-2"}, "evcc"},
		{[]string{"evcc"}, "evcc-2"},
		{[]string{"EVCC"}, "evcc-2"},
		{[]string{"evcc-2", "evcc"}, "evcc-3"},
		{[]string{"evcc", "evcc-3"}, "evcc-2"},
	} {
		assert.Equal(t, tc.expected, mdnsInstance(tc.taken), "taken: %v", tc.taken)
	}
}
