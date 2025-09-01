package tariff

import (
	"testing"
)

func TestTemplatesStekker(t *testing.T) {
	cfg := map[string]interface{}{
		"region": "BE",
	}

	s, err := NewStekkerFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create Stekker tariff provider: %v", err)
	}

	rates, err := s.Rates()
	if err != nil {
		t.Fatalf("failed to fetch rates: %v", err)
	}

	if len(rates) == 0 {
		t.Fatalf("no rates returned")
	}

	t.Logf("fetched %d rates, first=%+v", len(rates), rates[0])
}
