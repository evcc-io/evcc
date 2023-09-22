{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
features:
{{- if .integrateddevice }}
- integrateddevice
{{- end }}
{{- if .heating }}
- heating
{{- end }}
{{ if .icon }}icon: {{ .icon }}{{ end }}
{{ end -}}
