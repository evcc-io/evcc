{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
{{ if eq .integrateddevice "true" }}features: ["integrateddevice"]{{ end }}
{{ if ne .icon "" }}icon: {{ .icon }}{{ end }}
{{ end -}}
