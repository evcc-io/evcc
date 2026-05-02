{{- $prefix := .Function}}
func {{.Function}}(base {{.BaseType}}{{range orderedParams}}, {{.VarName}} {{.Signature}}{{end}}) {{.ReturnType}} {
	caps := make(map[reflect.Type]any)

	{{range groupedParams}}
	if {{.VarNames | first}} != nil {
		caps[reflect.TypeFor[{{.BaseType}}]()] = implement.{{.ShortType}}({{.VarNames | join ", "}})
	}
	{{end}}

	if len(caps) == 0 {
		return base
	}

	return &{{.Function}}Capable{ {{.ShortBase}}: base, caps: caps}
}

type {{.Function}}Capable struct {
	{{.BaseType}}
	caps map[reflect.Type]any
}

func (d *{{.Function}}Capable) Capability(typ reflect.Type) (any, bool) {
	c, ok := d.caps[typ]
	if !ok && reflect.TypeOf(d).Implements(typ) {
		return d, true
	}
	return c, ok
}
