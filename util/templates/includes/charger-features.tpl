{{ define "charger-features" }}
{{- if or (eq .heating "true") (eq .integrateddevice "true") }}
features:
{{- if eq .heating "true" }}
- heating
{{- end }}
{{- if eq .integrateddevice "true" }}
- integrateddevice
{{- end }}
{{- end }}
{{- end }}
