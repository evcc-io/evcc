{{ define "ocpp" }}
type: ocpp
{{- if ne .stationid "" }}
stationid: {{ .stationid }}
{{- end }}
{{- if ne .connector "1" }}
connector: {{ .connector }}
{{- end }}
{{- if ne .idtag "" }}
idtag: {{ .idtag }}
{{- end }}
connecttimeout: {{ .connecttimeout }}
timeout: {{ .timeout }}
{{- end }}
