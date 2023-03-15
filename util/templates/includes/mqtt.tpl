{{ define "mqtt" }}
source: mqtt
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
{{- end }}
