type: {{ .Type }}
{{ range .Params -}}
{{ .Name }}:
	{{- if len .Choice }} {{ join "|" .Choice }} {{- else }} {{ .Default }} {{- end }}
	{{- if len .Choice }} # <- choose one {{ .Name }} value {{- end }}
	{{- if .Hint }} # {{ .Hint }} {{- end }}
{{ end -}}
