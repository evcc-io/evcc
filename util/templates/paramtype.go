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
	_ ParamType = iota
	TypeString
	TypeNumber
	TypeFloat
	TypeBool
	TypeStringList
	TypeChargeModes
	TypeDuration
)
