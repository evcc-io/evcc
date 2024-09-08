{{ define "eebus" }}
type: eebus
ski: {{ .ski }}
{{ if .ip }}ip: {{ .ip }}{{ end }}
{{- end}}
