package provider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
)

type timeseriesProvider struct{}

func init() {
	registry.AddCtx("timeseries", TimeSeriesFromConfig)
}

// TimeSeriesFromConfig creates timeseries provider
func TimeSeriesFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	p := &timeseriesProvider{}

	return p, nil
}

var _ StringProvider = (*timeseriesProvider)(nil)

func (p *timeseriesProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		res := make(api.Rates, 48)
		ts := now.BeginningOfHour()
		for i := 0; i < 48; i++ {
			res[i] = api.Rate{
				Start: ts,
				End:   ts.Add(time.Hour),
			}
			ts = ts.Add(time.Hour)
		}

		b, err := json.Marshal(res)
		return string(b), err
	}, nil
}
