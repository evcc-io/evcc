package config

type Typed struct {
	Type  string                 `json:"type"`
	Other map[string]interface{} `mapstructure:",remain"`
}

type Named struct {
	Name  string                 `json:"name"`
	Type  string                 `json:"type"`
	Other map[string]interface{} `mapstructure:",remain"`
}
