package plugin

type GpioBias int

//go:generate go tool enumer -type GpioBias -trimprefix GpioBias -transform=kebab -text
const (
	GpioBiasAsIs GpioBias = iota
	GpioBiasDisabled
	GpioBiasPullUp
	GpioBiasPullDown
)
