{{ define "mqtt" }}
broker: {{ .host }}:{{ .port }}
{{- if .user }}
user: {{ .user }}
{{- end }}
{{- if .password }}
password: {{ .password }}
{{- end }}
{{- if ne .timeout "30s" }}
timeout: {{ .timeout }}
{{- end }}
{{- if ne .caCert "" }}
caCert: {{ .caCert }}
{{- end }}
{{- if ne .clientCert "" }}
clientCert: {{ .clientCert }}
{{- end }}
{{- if ne .clientKey "" }}
clientKey: {{ .clientKey }}
{{- end }}
{{- end }}
