package templates

type Class int

//go:generate go tool enumer -type Class -transform=lower
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
	Tariff
	Loadpoint
	Circuit
)
