package tariff

import "testing"

// TestStekkerAllZones tries all supported zones to ensure provider works.
func TestStekkerAllZones(t *testing.T) {
	for short, _ := range biddingZones {
		t.Run(short, func(t *testing.T) {
			p, err := NewStekkerFromConfig(map[string]interface{}{
				"zone": short,
			})
			if err != nil {
				t.Fatalf("zone %s config error: %v", short, err)
			}

			rates, err := p.Rates()
			if err != nil {
				t.Fatalf("zone %s fetch error: %v", short, err)
			}

			if len(rates) == 0 {
				t.Fatalf("zone %s returned no rates", short)
			}

			for _, r := range rates {
				if r.Price <= 0 {
					t.Errorf("zone %s invalid price: %+v", short, r)
				}
				if r.Start.Location().String() != "UTC" {
					t.Errorf("zone %s expected UTC, got %v", short, r.Start.Location())
				}
			}
		})
	}
}