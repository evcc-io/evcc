{{ define "ocpp20" }}
type: ocpp20
{{- if .stationid }}
stationid: {{ .stationid }}
{{- end }}
{{- if ne .evse "1" }}
evse: {{ .evse }}
{{- end }}
{{- if .idtag }}
idtag: {{ .idtag }}
{{- end }}
{{- if and .remotestart (ne .remotestart "false") }}
remotestart: {{ .remotestart }}
{{- end }}
{{- if and .meterinterval (ne .meterinterval "10s") }}
meterinterval: {{ .meterinterval }}
{{- end }}
{{- if ne .connecttimeout "5m" }}
connecttimeout: {{ .connecttimeout }}
{{- end }}
{{- end }}
