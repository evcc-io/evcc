{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
features:
{{- $integrateddeviceSet := and .integrateddevice (ne .integrateddevice "false") }}
{{- $heatingSet := and .heating (ne .heating "false") }}
{{- if $integrateddeviceSet }}
- integrateddevice
{{- end }}
{{- if $heatingSet }}
- heating
{{- end }}
{{- if .features }}
{{- range .features }}
{{- if eq . "integrateddevice" }}{{ if not $integrateddeviceSet }}
- {{ . }}
{{- end }}{{ else if eq . "heating" }}{{ if not $heatingSet }}
- {{ . }}
{{- end }}{{ else }}
- {{ . }}
{{- end }}
{{- end }}
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- end }}
