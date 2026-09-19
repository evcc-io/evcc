{{ define "features" }}
{{- if or .basefeatures .average .cacheable }}
features:
{{- range .basefeatures }}
- {{ . }}
{{- end }}
{{- if .average }}
- average
{{- end }}
{{- if and .cacheable (not (has "cacheable" .basefeatures)) }}
- cacheable
{{- end }}
{{- end }}
{{- end }}
