{{define "switchsocket"}}
{{ if eq .integrateddevice "true" }}features: ["integrateddevice"]{{ end }}
{{ if ne .icon "" }}icon: {{ .icon }}{{ end }}
{{end}}
