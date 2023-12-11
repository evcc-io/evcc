package modbus

type RegisterType int

//go:generate enumer -type RegisterType -trimprefix RegisterType -transform=lower
const (
	_ RegisterType = iota
	RegisterTypeInput
	RegisterTypeHolding
	RegisterTypeHoldings
	RegisterTypeCoil
	RegisterTypeCoils
)
