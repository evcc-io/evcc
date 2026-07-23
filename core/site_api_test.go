package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadpointsNilSlots(t *testing.T) {
	site := &Site{loadpoints: []*Loadpoint{new(Loadpoint), nil, new(Loadpoint)}}

	lps := site.Loadpoints()
	assert.Len(t, lps, 3, "disabled loadpoints must keep their slot")

	// disabled slot must be untyped nil, not a typed-nil interface
	assert.True(t, lps[1] == nil)
	assert.NotNil(t, lps[0])
	assert.NotNil(t, lps[2])

	assert.Len(t, site.activeLoadpoints(), 2)
	assert.True(t, site.IsConfigured())
}
