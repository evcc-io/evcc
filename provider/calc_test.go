package provider

import (
	"testing"
)

func TestCalcConfig(t *testing.T) {
	// must have at least one provider
	cc := map[string]interface{}{
		"add": []map[string]interface{}{},
	}

	_, err := NewCalcFromConfig(cc)
	if err == nil {
		t.Error(err)
	}

	// provider must be properly configured if specified
	cc = map[string]interface{}{
		"add": []map[string]interface{}{
			{},
		},
	}

	_, err = NewCalcFromConfig(cc)
	if err == nil {
		t.Error(err)
	}
}
