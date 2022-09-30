<!-- Detaillierte Problembeschreibung bitte hier -->



{{ if .CfgError -}}
Fehlermeldung:

```
{{ .CfgError }}
```

{{ end -}}

<details><summary>Konfiguration{{ if .CfgFile }} ({{ .CfgFile }}){{ end }}</summary>

```yaml
{{ .CfgContent }}
```

</details>

{{ if .Version -}}
Version: `{{ .Version }}`
{{ end -}}
