package plugin

import (
	"context"
	"errors"
	"testing"
)

func TestSolarmanFromConfig(t *testing.T) {
	provider, err := NewSolarmanFromConfig(t.Context(), map[string]any{
		"host":   "192.0.2.2",
		"serial": 1234567890,
		"register": map[string]any{
			"address": 86,
			"type":    "holding",
			"decode":  "uint32s",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := provider.(*Solarman).FloatGetter(); err != nil {
		t.Fatal(err)
	}
}

func TestSolarmanUsesConfigurationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	provider, err := NewSolarmanFromConfig(ctx, map[string]any{
		"host":   "192.0.2.2",
		"serial": 1234567890,
		"register": map[string]any{
			"address": 86,
			"type":    "holding",
			"decode":  "uint32s",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	getter, err := provider.(*Solarman).FloatGetter()
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	if _, err := getter(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got: %v", err)
	}
}
