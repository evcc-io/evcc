<!-- Detaillierte Problembeschreibung bitte hier -->



{{ if .CfgError -}}
Fehlermeldung:

```
{{ .CfgError }}
```

{{ end -}}

{{ if .CfgContent -}}
<details><summary>Konfiguration{{ if .CfgFile }} ({{ .CfgFile }}){{ end }}</summary>

```yaml
{{ .CfgContent }}
```

</details>
{{ end -}}

{{ if .Version -}}
Version: `{{ .Version }}`
{{ end -}}
