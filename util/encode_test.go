package util

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	assert.Equal(t, nil, EncodeAny(time.Time{}))
	assert.Equal(t, nil, EncodeAny(math.NaN()))
	assert.Equal(t, nil, EncodeAny(math.Inf(0)))
	assert.Equal(t, 3.142, EncodeAny(math.Pi))
}
