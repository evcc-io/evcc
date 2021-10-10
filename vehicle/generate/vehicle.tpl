type: {{ .Type }}
{{ range .Params -}}
{{ lower .Name }}:
{{- if ne .Label "" }} # {{ .Label }}{{ end }}
{{- if not .Required }} {{- if eq .Label "" }} #{{ end }} (optional){{ end }}
{{ end }}
