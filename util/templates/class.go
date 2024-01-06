package templates

type Class int

//go:generate enumer -type Class
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
	Tariff
)
