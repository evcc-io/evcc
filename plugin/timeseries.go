package plugin

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
func TimeSeriesFromConfig(_ context.Context, _ map[string]interface{}) (Plugin, error) {
	return new(timeseriesProvider), nil
}

var _ StringGetter = (*timeseriesProvider)(nil)

func (p *timeseriesProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		res := make(api.Rates, 48)
		ts := now.BeginningOfHour()
		for i := range 48 {
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
