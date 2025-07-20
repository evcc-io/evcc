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
{{- if .caCert }}
caCert: {{ .caCert }}
{{- end }}
{{- if .clientCert }}
clientCert: {{ .clientCert }}
{{- end }}
{{- if .clientKey }}
clientKey: {{ .clientKey }}
{{- end }}
{{- end }}
