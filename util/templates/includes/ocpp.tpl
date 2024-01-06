{{ define "ocpp" }}
type: ocpp
{{- if .stationid }}
stationid: {{ .stationid }}
{{- end }}
{{- if ne .connector "1" }}
connector: {{ .connector }}
{{- end }}
{{- if .idtag }}
idtag: {{ .idtag }}
{{- end }}
{{- if ne .connecttimeout "5m" }}
connecttimeout: {{ .connecttimeout }}
{{- end }}
{{- if ne .timeout "2m" }}
timeout: {{ .timeout }}
{{- end }}
{{- end }}
