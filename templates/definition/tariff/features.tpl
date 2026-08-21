{{ define "features" }}
{{- if or .basefeatures (eq .average "true") }}
features:
{{- range .basefeatures }}
- {{ . }}
{{- end }}
{{- if eq .average "true" }}
- average
{{- end }}
{{- end }}
{{- end }}
