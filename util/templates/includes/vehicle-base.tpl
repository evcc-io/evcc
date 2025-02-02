{{ define "vehicle-base" }}
user: {{ .user }}
password: {{ .password }}
vin: {{ .vin }}
{{ template "vehicle-common" . }}
{{- end }}
