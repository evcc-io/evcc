package custom

const (
	ParamUsage = "usage"
)

type Param struct {
	Name    string
	Default string
	Hint    string
	Choice  []string
}

type Template struct {
	Type   string
	Params []Param
	Render string // yaml rendering template
}

func (t *Template) Defaults() map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		if p.Default != "" {
			values[p.Name] = p.Default
		}
	}

	return values
}

func (t *Template) Usages() []string {
	for _, p := range t.Params {
		if p.Name == ParamUsage {
			return p.Choice
		}
	}
	return nil
}
