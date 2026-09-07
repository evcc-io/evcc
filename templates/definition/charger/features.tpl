{{ define "features" }}
{{- if or .heating .integrateddevice }}
features:
{{- if .heating }}
- heating
{{- end }}
{{- if .integrateddevice }}
- integrateddevice
{{- end }}
{{- end }}
{{- end }}
