package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Fixed struct {
	Price float64
}

var _ api.Tariff = (*Fixed)(nil)

func init() {
	registry.Add("fixed", NewFixedFromConfig)
}

func NewFixedFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc Fixed

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

// Rates implements the api.Tariff interface
func (t *Fixed) Rates() (api.Rates, error) {
	start := time.Now().Truncate(time.Hour)
	rr := api.Rates{{
		Start: start,
		End:   start.Add(time.Duration(24*7) * time.Hour),
		Price: t.Price,
	}}

	return rr, nil
}
