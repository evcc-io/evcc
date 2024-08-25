{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
features:
{{- if ne .integrateddevice "false" }}
- integrateddevice
{{- end }}
{{- if ne .heating "false" }}
- heating
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- end }}
