{{ define "features" }}
{{- if or .basefeatures .average }}
features:
{{- range .basefeatures }}
- {{ . }}
{{- end }}
{{- if .average }}
- average
{{- end }}
{{- end }}
{{- end }}
