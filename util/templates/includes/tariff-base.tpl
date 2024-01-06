{{ define "tariff-base" }}
{{- if .charges }}
charges: {{ .charges }}
{{- end }}
{{- if .tax }}
tax: {{ .tax }}
{{- end }}
{{- end }}
