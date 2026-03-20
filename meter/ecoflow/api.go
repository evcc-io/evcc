package ecoflow

import (
	"fmt"

	"github.com/spf13/cast"
)

// ExtractFloat extracts a float64 or int value from a map by key.
func ExtractFloat(data map[string]any, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return cast.ToFloat64E(v)
		}
	}
	return 0, api.ErrNotAvailable
}
