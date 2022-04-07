type: template
template: {{ .Template }}
{{- range .Params }}
{{- if or (ne (len .Value) 0) (ne (len .Values) 0) }} 
{{ .Name }}:
	{{- if len .Value }} {{ .Value }} {{ end }}
{{- if ne (len .Values) 0 }} 
{{- range .Values }}
- {{ . }}
{{- end }}
{{- end }}
{{- end -}}
{{ end -}}
