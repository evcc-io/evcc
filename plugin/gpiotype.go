package plugin

type GpioType int

//go:generate go tool enumer -type GpioType -trimprefix GpioType -transform=lower -text
const (
	GpioTypeRead GpioType = iota
	GpioTypeWrite
)
