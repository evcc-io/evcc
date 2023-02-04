package templates

type Class int

func (c *Class) UnmarshalText(text []byte) error {
	class, err := ClassString(string(text))
	if err == nil {
		*c = class
	}
	return err
}

//go:generate enumer -type Class
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
)
