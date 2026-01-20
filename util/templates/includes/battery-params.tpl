{{ define "battery-params" }}
{{- if gt .capacity 0 }}
capacity: {{ .capacity }} # kWh
{{- end }}
minsoc: {{ .minsoc }} # %
maxsoc: {{ .maxsoc }} # %
maxchargepower: {{ .maxchargepower }} # W
maxdischargepower: {{ .maxdischargepower }} # W
{{- end }}
