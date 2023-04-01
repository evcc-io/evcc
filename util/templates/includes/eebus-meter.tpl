{{ define "eebus-meter" }}
type: eebus
ski: {{ .ski }}
{{ if .ip }}ip: {{ .ip }}{{ end }}
meter: true
{{- end }}
