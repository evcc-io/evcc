{{ define "eebus-no-meter" }}
type: eebus
ski: {{ .ski }}
{{ if .ip }}ip: {{ .ip }}{{ end }}
{{- end}}
