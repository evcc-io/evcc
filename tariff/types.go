package tariff

type Typed struct {
	Type   string         `json:"type"`
	Tariff string         `json:"tariff"`
	Other  map[string]any `mapstructure:",remain" yaml:",inline"`
}

func (t Typed) Name() string {
	if t.Type == "template" {
		return t.Tariff
	}
	return t.Type
}
