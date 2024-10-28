{{ define "tariff-base" }}
{{- if .charges }}
charges: {{ .charges }}
{{- end }}
{{- if .tax }}
tax: {{ .tax }}
{{- end }}
{{- if .margin }}
margin: {{ .margin }}
{{- end }}
{{- if .uplifts }}
uplifts: {{ .uplifts }}
{{- end }}
{{- end }}
