{{ define "features" }}
{{- if or .basefeatures .coarsecurrent .welcomecharge .streaming .climaterdisabled .autodetectdisabled .wakeupdisabled }}
features:
{{- range .basefeatures }}
- {{ . }}
{{- end }}
{{- if .coarsecurrent }}
- coarsecurrent
{{- end }}
{{- if .welcomecharge }}
- welcomecharge
{{- end }}
{{- if .streaming }}
- streaming
{{- end }}
{{- if .climaterdisabled }}
- climaterdisabled
{{- end }}
{{- if .autodetectdisabled }}
- autodetectdisabled
{{- end }}
{{- if .wakeupdisabled }}
- wakeupdisabled
{{- end }}
{{- end }}
{{- end }}
