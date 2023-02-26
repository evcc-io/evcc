{{define "eebus-no-meter"}}
type: eebus
ski: {{ .ski }}
{{ if ne .ip "" }}ip: {{ .ip }}{{ end }}
{{end}}
