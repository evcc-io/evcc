<!-- Detaillierte Problembeschreibung bitte hier -->



{{ if .CfgError -}}
Fehlermeldung:

    {{ .CfgError }}

{{ end -}}

<details><summary>Konfiguration:</summary>

```
{{ .CfgContent }}
```

</details>

{{ if .CfgFile -}}
Pfad: `{{ .CfgFile }}`
{{ end -}}

{{ if .Version -}}
Version: `{{ .Version }}`
{{ end -}}
