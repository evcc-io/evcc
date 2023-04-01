{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
{{ if eq .integrateddevice "true" }}features: ["integrateddevice"]{{ end }}
{{ if .icon }}icon: {{ .icon }}{{ end }}
{{ end -}}
