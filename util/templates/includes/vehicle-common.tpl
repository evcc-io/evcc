{{ define "vehicle-common" }}
{{- if .title }}
title: {{ .title }}
{{- end }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- if .capacity }}
capacity: {{ .capacity }}
{{- end }}
{{- if .phases }}
phases: {{ .phases }}
{{- end }}

{{- if or .mode .minCurrent .maxCurrent .priority }}
onIdentify:
{{- if .mode }}
  mode: {{ .mode }}
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
{{- range .identifiers }}
- {{ . }}
{{- end }}
{{- end }}

{{- end }}
