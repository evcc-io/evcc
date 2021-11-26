type: template
template: {{ .Template }}
{{- if .Description }}
description: {{ .Description }}
{{- end }}
{{ range .Params -}}
{{ .Name }}:
	{{- if len .Value }} {{ .Value }} {{ end }}
	{{- if .Help.DE }} # {{ .Help.DE }} {{- end }}
{{ end -}}
