{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
{{ if and .integrateddevice .heating }}
features: ["integrateddevice", "heating"]
{{ else if .integrateddevice }}
features: ["integrateddevice"]
{{ else if .heating }}
features: ["heating"]
{{ end }}
{{ if .icon }}icon: {{ .icon }}{{ end }}
{{ end -}}
