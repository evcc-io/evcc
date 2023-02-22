package templates

type ParamType int

func (c *ParamType) UnmarshalText(text []byte) error {
	typ, err := ParamTypeString(string(text))
	if err == nil {
		*c = typ
	}
	return err
}

//go:generate enumer -type ParamType -trimprefix Type
const (
	TypeString ParamType = iota // default type string
	TypeBool
	TypeChoice
	TypeChargeModes
	TypeDuration
	TypeFloat
	TypeNumber
	TypeStringList
)
