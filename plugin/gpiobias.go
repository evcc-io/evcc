package plugin

// GpioBias is the internal pull resistor configuration of a GPIO line
type GpioBias int

//go:generate go tool enumer -type GpioBias -trimprefix GpioBias -transform=kebab -text
const (
	GpioBiasAsIs     GpioBias = iota // GpioBiasAsIs leaves the bias unchanged
	GpioBiasDisabled                 // GpioBiasDisabled disables the internal pull resistor
	GpioBiasPullUp                   // GpioBiasPullUp enables the internal pull-up resistor
	GpioBiasPullDown                 // GpioBiasPullDown enables the internal pull-down resistor
)
