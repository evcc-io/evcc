{{ define "tariff-base" }}
{{- if .costs }}
charges: {{ .costs }}
{{- end }}
{{- if .tax }}
tax: {{ .tax }}
{{- end }}
{{- end }}
