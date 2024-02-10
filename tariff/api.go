package tariff

import (
	"github.com/evcc-io/evcc/api"
	"golang.org/x/text/currency"
)

type API interface {
	GetCurrency() currency.Unit
	SetCurrency(string) error

	GetRef(ref string) string
	SetRef(ref, value string)

	GetInstance(ref string) api.Tariff
	SetInstance(ref string, tariff api.Tariff)

	CurrentGridPrice() (float64, error)
	CurrentFeedInPrice() (float64, error)
	CurrentCo2() (float64, error)
}
