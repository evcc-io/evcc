{{ define "tariff-base" }}
{{- if .charges }}
charges: {{ .charges }}
{{- end }}
{{- if .tax }}
tax: {{ .tax }}
{{- end }}
{{- if .formula }}
formula: {{ .formula }}
{{- end }}
{{- end }}
