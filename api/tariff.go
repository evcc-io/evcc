package api

import "fmt"

type TariffType int

const (
	TariffTypePrice TariffType = iota
	TariffTypeCo2
)

func (t TariffType) String() string {
	switch t {
	case TariffTypePrice:
		return "price"
	case TariffTypeCo2:
		return "co2"
	default:
		return fmt.Sprintf("Unknown TariffType (%d)", t)
	}
}
