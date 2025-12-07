package plugin

type GpioType int

//go:generate go tool enumer -type GpioType -trimprefix GpioType -transform=lower
const (
	GpioTypeRead GpioType = iota
	GpioTypeWrite
)
