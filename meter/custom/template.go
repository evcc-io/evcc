package custom

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
