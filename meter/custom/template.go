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
	// Sample string // yaml sample for README
	Render string // yaml rendering template
}

func (t *Template) Usages() []string {
	for _, p := range t.Params {
		if p.Name == ParamUsage {
			return p.Choice
		}
	}
	return nil
}
