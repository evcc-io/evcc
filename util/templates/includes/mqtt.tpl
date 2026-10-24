{{ define "mqtt" }}
{{- if contains ":" .host }}
broker: {{ .host }}
{{- else if .host }}
broker: {{ joinHostPort .host .port }}
{{- end }}
{{- if .user }}
user: {{ .user }}
{{- end }}
{{- if .password }}
password: {{ .password }}
{{- end }}
{{- if .timeout }}
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
