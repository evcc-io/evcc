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

{{ define "battery-params" }}
{{- include "battery-capacity" . }}
{{- include "battery-minmaxsoc" . }}
{{- include "battery-power" . }}
{{- end }}
