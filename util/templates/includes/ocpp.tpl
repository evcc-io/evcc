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
connecttimeout: {{ .connecttimeout }}
timeout: {{ .timeout }}
{{- end }}
