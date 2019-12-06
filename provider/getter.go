package provider

// Getters return typed device data.
// They are used to abstract the underlying device implementation.

type (
	// FloatGetter gets float value
	FloatGetter func() (float64, error)
	// IntGetter gets int value
	IntGetter func() (int64, error)
	// StringGetter gets string value
	StringGetter func() (string, error)
	// BoolGetter gets bool value
	BoolGetter func() (bool, error)
)
