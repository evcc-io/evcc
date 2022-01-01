package tariff

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Fixed struct {
	Price float64
}

var _ api.Tariff = (*Fixed)(nil)

func NewFixed(other map[string]interface{}) (*Fixed, error) {
	cc := Fixed{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

func (t *Fixed) CurrentPrice() (float64, error) {
	return t.Price, nil
}

func (t *Fixed) IsCheap() (bool, error) {
	return false, nil
}
