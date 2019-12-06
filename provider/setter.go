package provider

// Setters update typed device data.
// They are used to abstract the underlying device implementation.

type (
	// FloatSetter sets float value
	FloatSetter func(float64) error
	// IntSetter sets int value
	IntSetter func(int64) error
	// StringSetter sets string value
	StringSetter func(string) error
	// BoolSetter sets bool value
	BoolSetter func(bool) error
)
