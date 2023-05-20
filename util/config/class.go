package config

type Class int

//go:generate enumer -type Class
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
)
