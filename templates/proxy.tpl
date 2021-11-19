type: {{ .Template }}
{{- if .Description }}
description: {{ .Description }}
{{- end }}
{{ range .Params -}}
{{ .Name }}:
	{{- if len .Value }}{{if or (eq .ValueType "") (eq .ValueType "string") }} "{{ .Value }}" {{ else }} {{ .Value }}{{ end }}{{ end }}
	{{- if .Help.DE }} # {{ .Help.DE }} {{- end }}
{{ end -}}
