package provider

// IntProvider reads int values
type IntProvider interface {
	IntGetter() func() (int64, error)
}

// StringProvider reads string values
type StringProvider interface {
	StringGetter() func() (string, error)
}

// FloatProvider reads float values
type FloatProvider interface {
	FloatGetter() func() (float64, error)
}

// BoolProvider reads bool values
type BoolProvider interface {
	BoolGetter() func() (bool, error)
}

// SetIntProvider writes int values
type SetIntProvider interface {
	IntSetter(param string) func(int64) error
}

// SetBoolProvider writes bool values
type SetBoolProvider interface {
	BoolSetter(param string) func(bool) error
}
