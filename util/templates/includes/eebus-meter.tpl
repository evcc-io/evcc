{{ define "eebus-meter" }}
type: eebus
ski: {{ .ski }}
{{ if ne .ip "" }}ip: {{ .ip }}{{ end }}
meter: true
{{- end }}
