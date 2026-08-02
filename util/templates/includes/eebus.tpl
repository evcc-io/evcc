{{ define "eebus" }}
ski: {{ .ski }}
{{ if .ip }}ip: {{ .ip }}{{ end }}
{{- end}}
