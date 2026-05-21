{{ define "vehicle-features" }}
{{- if or (eq .coarsecurrent "true") (eq .welcomecharge "true") (eq .streaming "true") (eq .disablechargingonclimateractive "true")}}
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
{{- if eq .disablechargingonclimateractive "true" }}
- disablechargingonclimateractive
{{- end }}
{{- end }}
{{- end }}
