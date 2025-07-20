package tesla

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiError(t *testing.T) {
	assert.Nil(t, apiError(nil))
}
