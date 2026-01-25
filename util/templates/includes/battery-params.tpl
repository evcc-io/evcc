{{ define "battery-params" }}
capacity: {{ .capacity }} # kWh
minsoc: {{ .minsoc }} # %
maxsoc: {{ .maxsoc }} # %
maxchargepower: {{ .maxchargepower }} # W
maxdischargepower: {{ .maxdischargepower }} # W
{{- end }}
