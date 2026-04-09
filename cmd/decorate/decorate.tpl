{{- $prefix := .Function}}
func {{.Function}}(base {{.BaseType}}{{range orderedParams}}, {{.VarName}} {{.Signature}}{{end}}) {{.ReturnType}} {
	caps := make(map[reflect.Type]any)

	{{range orderedParams}}
	if {{.VarName}} != nil {
		caps[reflect.TypeFor[{{.BaseType}}]()] = &{{$prefix}}{{.ShortType}}Impl{ {{.VarName}}: {{.VarName}} }	
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
	return c, ok
}

{{range $element := .Types -}}
type {{$prefix}}{{$element.ShortType}}Impl struct {
	{{- range .Functions}}
	{{.VarName}} {{.Signature}}
	{{- end}}
}
{{range $element.Functions}}
func (impl *{{$prefix}}{{$element.ShortType}}Impl) {{.Function}}(
	{{- range $idx, $param := .Params -}}
		{{- if gt $idx 0}}, {{end -}}
		p{{$idx}} {{ $param -}} 
	{{end}}){{ .ReturnTypes }} {
	return impl.{{.VarName}}(
	{{- range $idx, $param := .Params -}}
		{{- if gt $idx 0}}, {{end}}p{{- $idx -}}{{end}})
}
{{end}}
{{end}}
