{{ define "tariff-features" }}
{{- if eq .average "true" }}
features: ["average"]
{{- end }}
{{- end }}
