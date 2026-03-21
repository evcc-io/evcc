func {{.Function}}(base {{.BaseType}}{{range ordered}}, {{.VarName}} {{.Signature}}{{end}}) {{.ReturnType}} {
{{- $basetype := .BaseType}}
{{- $shortbase := .ShortBase}}
{{- $prefix := .Function}}
{{- $types := .Types}}
{{- $and := false}}
	caps := make(map[reflect.Type]any)

{{range $api, $element := .Types}}
	// {{$api}}, {{$element}}
	{{range $func := $element.Functions}}
		if {{.VarName}} != nil {
			caps[reflect.TypeFor[{{$api}}]()] = {{.VarName}}
		}
	{{end}}
{{end}}

	return &{{.Function}}Capable{
		caps: caps,
		{{.ReturnType}}: base,
	}
}

type {{.Function}}Capable struct {
	{{.ReturnType}}
	caps map[reflect.Type]any
}

func (d *{{.Function}}Capable) Capability(typ reflect.Type) (any, bool) {
	return d.caps[typ]
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
