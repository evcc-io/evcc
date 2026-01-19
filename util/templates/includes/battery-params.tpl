{{ define "battery-params" }}
{{- if .capacity }}
capacity: {{ .capacity }} # kWh
{{- end }}
minsoc: {{ .minsoc }} # %
maxsoc: {{ .maxsoc }} # %
maxchargepower: {{ .maxchargepower }} # W
maxdischargepower: {{ .maxdischargepower }} # W
{{- end }}
