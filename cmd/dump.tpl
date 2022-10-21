
{{ if .CfgError -}}
Fehlermeldung:

{{ .CfgError | indent 4 }}

{{ end -}}

{{ if .CfgContent -}}
Konfiguration{{ if .CfgFile }} ({{ .CfgFile }}){{ end }}:

{{ .CfgContent }}

{{ end -}}

{{ if .Version -}}
Version: `{{ .Version }}`
{{ end -}}
