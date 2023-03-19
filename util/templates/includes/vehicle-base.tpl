{{ define "vehicle-base" }}
{{- if ne .title "" }}
title: {{ .title }}
{{- end }}
{{- if ne .icon "" }}
icon: {{ .icon }}
{{- end }}
user: {{ .user }}
password: {{ .password }}
{{- if ne .capacity "" }}
capacity: {{ .capacity }}
{{- end }}
{{- if ne .vin "" }}
vin: {{ .vin }}
{{- end }}
{{- if ne .phases "" }}
phases: {{ .phases }}
{{- end }}
{{- if ne .cache "" }}
cache: {{ .cache }}
{{- end }}
{{- end }}
