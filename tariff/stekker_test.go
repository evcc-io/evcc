package tariff

import (
	"testing"
)

func TestStekkerRates(t *testing.T) {
	s, err := NewStekkerFromConfig(map[string]interface{}{
		"zone": "BE",
	})
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	rates, err := s.Rates()
	if err != nil {
		t.Fatalf("failed to get rates: %v", err)
	}

	if len(rates) == 0 {
		t.Fatalf("no rates returned")
	}

	t.Logf("fetched %d rates, first=%+v", len(rates), rates[0])
}
