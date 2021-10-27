type: {{ .Template }}
{{ range .Params -}}
{{ .Name }}:
  {{- if len .Value }} {{ .Value }} {{ else }}
	{{- if len .Choice }} {{ join "|" .Choice }} {{- else }} {{ .Default }} {{- end }}
	{{- if len .Choice }} # <- choose one {{ .Name }} value {{- end }}
	{{- end }}
	{{- if .Hint }} # {{ .Hint }} {{- end }}
{{ end -}}
