package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Temperature struct {
	Temp      *plugin.Config // optional
	LimitTemp *plugin.Config // optional
}

func (cc *Temperature) Configure(ctx context.Context) (
	func() (float64, error),
	func() (int64, error),
	error,
) {
	tempG, err := cc.Temp.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("temp: %w", err)
	}

	limitTempG, err := cc.LimitTemp.IntGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("limit temp: %w", err)
	}

	return tempG, limitTempG, nil
}
