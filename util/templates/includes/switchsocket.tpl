{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
features:
{{- if and .integrateddevice (ne .integrateddevice "false") }}
- integrateddevice
{{- end }}
{{- if and .heating (ne .heating "false") }}
- heating
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- end }}
