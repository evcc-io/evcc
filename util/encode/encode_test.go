package encode

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	enc := NewEncoder(WithDuration())
	assert.Equal(t, nil, enc.Encode(time.Time{}))
	assert.Equal(t, nil, enc.Encode(math.NaN()))
	assert.Equal(t, nil, enc.Encode(math.Inf(0)))
	assert.Equal(t, 30, enc.Encode(30*time.Second))
	assert.Equal(t, 3.142, enc.Encode(math.Pi))
}
