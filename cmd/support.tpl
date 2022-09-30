<!-- Detaillierte Problembeschreibung bitte hier -->



{{ if .CfgError -}}
Fehlermeldung:

    {{ .CfgError }}

{{ end -}}

<details><summary>Konfiguration:</summary>

```yaml
{{ .CfgContent }}
```

</details>

{{ if .CfgFile -}}
Pfad: `{{ .CfgFile }}`
{{ end -}}

{{ if .Version -}}
Version: `{{ .Version }}`
{{ end -}}
