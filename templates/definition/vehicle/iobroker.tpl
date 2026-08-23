{{ define "iobroker-get" }}
{{ .name }}:
  source: http
  uri: {{ .uri }}/rest-api/v1/state/{{ .state }}/plain?extraPlain=true
  headers:
    - accept: text/plain;charset=UTF-8
  cache: {{ .cache }}
{{- if .jq }}
  quote: true
  jq: {{ .jq }}
{{- end }}
{{- if .user }}
  auth:
    type: basic
    user: {{ .user }}
    password: {{ .password }}
{{- end }}
{{- end }}

{{ define "iobroker-set" }}
{{ .name }}:
  source: http
  uri: {{ .uri }}/rest-api/v1/state/{{ .state }}?value={{ .value }}
  headers:
    - accept: text/plain;charset=UTF-8
{{- if .user }}
  auth:
    type: basic
    user: {{ .user }}
    password: {{ .password }}
{{- end }}
{{- end }}
