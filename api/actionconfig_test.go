package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	pv := ModePV
	one := 1
	six := 6.0
	old := ActionConfig{
		Mode:       &pv,
		Priority:   &one,
		MinCurrent: &six,
	}

	now := ModeNow
	two := 2
	three := 3
	new := ActionConfig{
		Mode:     &now,
		MinSoc:   &three,
		Priority: &two,
	}

	dst := old.Merge(new)

	// unmodified
	assert.Equal(t, old, ActionConfig{
		Mode:       &pv,
		Priority:   &one,
		MinCurrent: &six,
	}, "old modified")

	// overwritten
	assert.Equal(t, dst, ActionConfig{
		Mode:       &now,
		MinCurrent: &six,
		MinSoc:     &three,
		Priority:   &two,
	}, "new wrong")
}
