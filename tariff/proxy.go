package tariff

import (
	"context"
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// proxyFeatures are handled by the proxies instead of the tariff itself
var proxyFeatures = []api.Feature{api.Average, api.Cacheable}

// NewProxyFromConfig creates a tariff proxy supporting average and caching
func NewProxyFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	var embed struct {
		Features []api.Feature  `mapstructure:"features"`
		Other    map[string]any `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &embed); err != nil {
		return nil, err
	}

	isProxyFeature := func(f api.Feature) bool {
		return slices.Contains(proxyFeatures, f)
	}

	if !slices.ContainsFunc(embed.Features, isProxyFeature) {
		return NewFromConfig(ctx, typ, other)
	}

	// pass remaining features on to the tariff
	if features := slices.DeleteFunc(slices.Clone(embed.Features), isProxyFeature); len(features) > 0 {
		embed.Other["features"] = features
	} else {
		delete(embed.Other, "features")
	}

	// caching is the inner proxy to persist the tariff's original rates
	newTariff := NewFromConfig
	if slices.Contains(embed.Features, api.Cacheable) {
		newTariff = NewCachedFromConfig
	}

	t, err := newTariff(ctx, typ, embed.Other)
	if err != nil {
		return nil, err
	}

	if slices.Contains(embed.Features, api.Average) {
		return NewAverageProxy(t)
	}

	return t, nil
}
