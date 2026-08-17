{{ define "heatpumpswitch" }}
features:
- continuous
{{- if eq .tempsource "warmwater" }}
- predictorprofileweekday
{{- else }}
- predictorprofiletemperature
{{- end }}
- heating
- integrateddevice
- switchdevice
{{- end }}
