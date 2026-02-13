package modbus

// go:generate go tool enumer -type ReadOnlyMode -trimprefix ReadOnly -transform=lower

type ReadOnlyMode int

const (
	ReadOnlyFalse ReadOnlyMode = iota
	ReadOnlyDeny               // return modbus error
	ReadOnlyTrue               // silently ignore writes
)
