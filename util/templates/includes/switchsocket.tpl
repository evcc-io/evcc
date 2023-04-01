{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
{{ if .integrateddevice }}features: ["integrateddevice"]{{ end }}
{{ if .icon }}icon: {{ .icon }}{{ end }}
{{ end -}}
