{{ define "heatpumpswitch" }}
features:
- continuous
{{- if eq .tempsource "warmwater" }}
- demandprofilesameweekday
{{- else }}
- demandprofiledailytemperature
{{- end }}
- heating
- integrateddevice
- switchdevice
{{- end }}
