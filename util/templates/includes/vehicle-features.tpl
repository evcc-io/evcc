{{ define "vehicle-features" }}
{{- if or (eq .coarsecurrent "true") (eq .welcomecharge "true") }}
features:
{{- if eq .coarsecurrent "true" }}
- coarsecurrent
{{- end }}
{{- if eq .welcomecharge "true" }}
- welcomecharge
{{- end }}
{{- end }}
{{- end }}
