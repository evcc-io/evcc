{{ define "vehicle-base" }}
{{- if .title }}
title: {{ .title }}
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
user: {{ .user }}
password: {{ .password }}
{{- if .capacity }}
capacity: {{ .capacity }}
{{- end }}
{{- if .vin }}
vin: {{ .vin }}
{{- end }}
{{- if .phases }}
phases: {{ .phases }}
{{- end }}
{{- if .cache }}
cache: {{ .cache }}
{{- end }}
{{- end }}
