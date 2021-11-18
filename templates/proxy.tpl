type: {{ .Template }}
description: {{ .Description }}
{{ range .Params -}}
{{ .Name }}:
	{{- if len .Value }}{{if or (eq .ValueType "") (eq .ValueType "string") }} "{{ .Value }}" {{ else }} {{ .Value }}{{ end }}{{ end }}
	{{- if .Help }} # {{ .Help }} {{- end }}
{{ end -}}
