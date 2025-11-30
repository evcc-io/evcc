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

type FromTo struct {
	From, To int
}

func (ft FromTo) IsActive(hour int) bool {
	return ft.From == 0 && ft.To == 0 ||
		ft.From < ft.To && ft.From <= hour && hour <= ft.To ||
		ft.From > ft.To && (ft.From <= hour || hour <= ft.To)
}
