{{ define "vehicle-features" }}
{{- if or (eq .coarsecurrent "true") (eq .welcomecharge "true") (eq .streaming "true") (eq .climaterdisabled "true") }}
features:
{{- if eq .coarsecurrent "true" }}
- coarsecurrent
{{- end }}
{{- if eq .welcomecharge "true" }}
- welcomecharge
{{- end }}
{{- if eq .streaming "true" }}
- streaming
{{- end }}
{{- if eq .climaterdisabled "true" }}
- climaterdisabled
{{- end }}
{{- end }}
{{- end }}
