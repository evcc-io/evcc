{{ define "base" }}
user: {{ .user }}
password: {{ .password }}
vin: {{ .vin }}
{{ template "common" . }}
{{- if .cache }}
cache: {{ .cache }}
{{- end }}
{{- end }}
