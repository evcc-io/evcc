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
{{- range .features }}
{{- if and (ne . "integrateddevice") (ne . "heating") }}
- {{ . }}
{{- else if and (eq . "integrateddevice") (not $integrateddeviceSet) }}
- {{ . }}
{{- else if and (eq . "heating") (not $heatingSet) }}
- {{ . }}
{{- end }}
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- end }}
