package provider

import (
	"testing"
)

func TestModbusConfig(t *testing.T) {
	// must have at least one provider
	cc := map[string]interface{}{
		"model":    "foo",
		"id":       1,               // squashed settings
		"uri":      "localhost:502", // squashed settings
		"register": map[string]interface{}{},
		"value":    "power",
	}

	_, err := NewModbusFromConfig(cc)
	if err != nil {
		t.Error(err)
	}
}
