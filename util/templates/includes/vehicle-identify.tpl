{{ define "vehicle-identify" }}
{{- if or .mode .minSoc .targetSoc .minCurrent .maxCurrent .priority }}
onIdentify:
{{- if .mode }}
  mode: {{ .mode }}
{{- end }}
{{- if .minSoc }}
  minSoc: {{ .minSoc }}
{{- end }}
{{- if .targetSoc }}
  targetSoc: {{ .targetSoc }}
{{- end }}
{{- if .minCurrent }}
  minCurrent: {{ .minCurrent }}
{{- end }}
{{- if .maxCurrent }}
  maxCurrent: {{ .maxCurrent }}
{{- end }}
{{- if .priority }}
  priority: {{ .priority }}
{{- end }}
{{- end }}
{{- if len .identifiers }}
identifiers:
{{-   range .identifiers }}
- {{ . }}
{{-   end }}
{{- end }}
{{- end }}
