{{ define "battery-capacity" }}
capacity: {{ .capacity }} # kWh
{{- end }}

{{ define "battery-minmaxsoc" }}
minsoc: {{ .minsoc }} # %
maxsoc: {{ .maxsoc }} # %
{{- end }}

{{ define "battery-power" }}
maxchargepower: {{ .maxchargepower }} # W
maxdischargepower: {{ .maxdischargepower }} # W
{{- end }}

{{/* connection for templates without the modbus preset, i.e. fixed sunspec device id 1 */}}
{{ define "battery-sunspec-connection" }}
{{- if .modbus }}
{{- include "modbus" . }}
{{- else }}
uri: {{ joinHostPort .host .port }}
id: 1
{{- end }}
{{- end }}

{{ define "battery-power-sunspec" }}
maxchargepower:
  source: sunspec
  {{- include "battery-sunspec-connection" . | indent 2 }}
  value:
    - 120:0:MaxChaRte
    - 802:WChaRteMax
maxdischargepower:
  source: sunspec
  {{- include "battery-sunspec-connection" . | indent 2 }}
  value:
    - 120:0:MaxDisChaRte
    - 802:WDisChaRteMax
{{- end }}

{{ define "battery-params" }}
{{- include "battery-capacity" . }}
{{- include "battery-minmaxsoc" . }}
{{- include "battery-power" . }}
{{- end }}

{{ define "battery-params-sunspec" }}
{{- include "battery-capacity" . }}
{{- include "battery-minmaxsoc" . }}
{{- include "battery-power-sunspec" . }}
{{- end }}
