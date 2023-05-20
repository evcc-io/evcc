package config

type container[T any] struct {
	config Named
	device T
}

type Typed struct {
	Type  string                 `json:"type"`
	Other map[string]interface{} `mapstructure:",remain"`
}

type Named struct {
	Name  string                 `json:"name"`
	Type  string                 `json:"type"`
	Other map[string]interface{} `mapstructure:",remain"`
}
