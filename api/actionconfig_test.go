package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionConfigString(t *testing.T) {
	var a ActionConfig
	assert.NotPanics(t, func() {
		_ = a.String()
	})
}
