package ecoflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigDecode(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		var c config
		input := map[string]any{
			"uri": "http://test",
			"sn":  "123",
		}
		
		err := c.decode(input)
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, c.Cache)
		assert.Equal(t, "http://test", c.URI)
	})

	t.Run("override cache", func(t *testing.T) {
		var c config
		input := map[string]any{
			"cache": "1m",
		}
		
		err := c.decode(input)
		assert.NoError(t, err)
		assert.Equal(t, time.Minute, c.Cache)
	})
}
