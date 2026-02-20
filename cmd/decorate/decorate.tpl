{{define "case"}}
	{{- $combo := .Combo}}
	{{- $prefix := .Prefix}}
	{{- $and := false}}

	{{- range $typ, $def := .Types}}
		{{- if requiredType $combo $typ}}
			{{- range $def.Functions}}
				{{- if $and}} &&{{else}}{{$and = true}}{{end}} {{.VarName}} {{if contains $combo $typ}}!={{else}}=={{end}} nil
			{{- end}}
		{{- end}}
	{{- end}}:
		return &struct {
			{{.BaseType}}
{{- range $typ, $def := .Types}}
	{{- if contains $combo $typ}}
			{{$typ}}
	{{- end}}
{{- end}}
		}{
			{{.ShortBase}}: base,
{{- range $typ, $def := .Types}}
	{{- if contains $combo $typ}}
			{{$def.ShortType}}: &{{$prefix}}{{$def.ShortType}}Impl{
				{{- range $def.Functions}}
				{{.VarName}}: {{.VarName}},
				{{- end}}
			},
	{{- end}}
{{- end}}
		}
{{- end -}}

func {{.Function}}(base {{.BaseType}}{{range ordered}}, {{.VarName}} {{.Signature}}{{end}}) {{.ReturnType}} {
{{- $basetype := .BaseType}}
{{- $shortbase := .ShortBase}}
{{- $prefix := .Function}}
{{- $types := .Types}}
{{- $and := false}}
	switch {
	case {{- range $typ, $def := .Types}}
		{{- if requiredType empty $typ}}
			{{- range $def.Functions}}
				{{- if $and}} &&{{else}}{{$and = true}}{{end}} {{.VarName}} == nil
			{{- end}}
		{{- end}}
	{{- end}}:
		return base
{{range $combo := .Combinations}}
	case {{- template "case" dict "BaseType" $basetype "Prefix" $prefix "ShortBase" $shortbase "Types" $types "Combo" $combo}}
{{end}}	}

	return nil
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
