{{define "renderbase-vehicle"}}
{{- if ne .title "" }}
title: {{ .title }}
{{- end }}
user: {{ .user }}
password: {{ .password }}
{{- if ne .capacity "" }}
capacity: {{ .capacity }}
{{- end }}
{{- if ne .vin "" }}
vin: {{ .vin }}
{{- end }}
{{- if ne (len .identifiers) 0 }}
identifiers:
{{-   range .identifiers }}
- {{ . }}
{{-   end }}
{{- end }}
{{end}}
