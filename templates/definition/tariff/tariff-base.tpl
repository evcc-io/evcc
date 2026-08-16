{{ define "tariff-base" }}
{{- if .charges }}
charges: {{ .charges }}
{{- end }}
{{- if .chargesZones }}
chargesZones:
{{- range .chargesZones }}
  - charges: {{ .price }}
    {{- if .days }}
    days: {{ .days }}
    {{- end }}
    {{- if .hours }}
    hours: {{ .hours }}
    {{- end }}
    {{- if .months }}
    months: {{ .months }}
    {{- end }}
{{- end }}
{{- end }}
{{- if .tax }}
tax: {{ .tax }}
{{- end }}
{{- if .formula }}
formula: {{ .formula }}
{{- end }}
{{- end }}
