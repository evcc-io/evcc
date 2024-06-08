{{ define "vehicle-base" }}
user: {{ .user }}
password: {{ .password }}
{{ template "vehicle-common" . }}
{{- end }}
