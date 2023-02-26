{{define "vehicle-identify"}}
{{- if or (ne .mode "") (ne .minSoc "") (ne .targetSoc "") (ne .minCurrent "") (ne .maxCurrent "") (ne .priority "") }}
onIdentify:
{{- if (ne .mode "") }}
  mode: {{ .mode }}
{{- end }}
{{- if (ne .minSoc "") }}
  minSoc: {{ .minSoc }}
{{- end }}
{{- if (ne .targetSoc "") }}
  targetSoc: {{ .targetSoc }}
{{- end }}
{{- if (ne .minCurrent "") }}
  minCurrent: {{ .minCurrent }}
{{- end }}
{{- if (ne .maxCurrent "") }}
  maxCurrent: {{ .maxCurrent }}
{{- end }}
{{- if (ne .priority "") }}
- priority: {{ .priority }}
{{- end }}
{{- end }}
{{- if ne (len .identifiers) 0 }}
identifiers:
{{-   range .identifiers }}
- {{ . }}
{{-   end }}
{{- end }}
{{end}}
