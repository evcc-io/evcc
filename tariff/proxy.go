package tariff

import (
	"context"
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// NewProxyFromConfig creates a tariff proxy supporting average or caching
func NewProxyFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	var embed struct {
		Features []api.Feature  `mapstructure:"features"`
		Other    map[string]any `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &embed); err != nil {
		return nil, err
	}

	if slices.Contains(embed.Features, api.Average) {
		embed.Other["features"] = sliceMinusElement(embed.Features, api.Average)
		t, err := NewFromConfig(ctx, typ, embed.Other)
		if err != nil {
			return nil, err
		}
		return NewAverageProxy(t)
	}

	if slices.Contains(embed.Features, api.Cacheable) {
		embed.Other["features"] = sliceMinusElement(embed.Features, api.Cacheable)
		return NewCachedFromConfig(ctx, typ, embed.Other)
	}

	return NewFromConfig(ctx, typ, other)
}
