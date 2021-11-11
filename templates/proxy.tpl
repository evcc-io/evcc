type: {{ .Template }}
{{ range .Params -}}
{{ .Name }}:
  {{- if len .Value }} {{if or (eq .ValueType "") (eq .ValueType "string") }}'{{end}}{{ .Value }}{{if or (eq .ValueType "") (eq .ValueType "string") }}'{{end}} {{ else }}
	{{- if len .Choice }} {{ join "|" .Choice }} {{- else }} {{ .Default }} {{- end }}
	{{- if len .Choice }} # <- choose one {{ .Name }} value {{- end }}
	{{- end }}
	{{- if .Help }} # {{ .Help }} {{- end }}
{{ end -}}
