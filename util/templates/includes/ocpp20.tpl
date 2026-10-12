{{ define "ocpp20" }}
type: ocpp20
{{- if .stationid }}
stationid: {{ .stationid }}
{{- end }}
{{- if ne .evse "1" }}
evse: {{ .evse }}
{{- end }}
{{- if ne .connector "1" }}
connector: {{ .connector }}
{{- end }}
{{- if .idtag }}
idtag: {{ .idtag }}
{{- end }}
{{- if and .remotestart (ne .remotestart "false") }}
remotestart: {{ .remotestart }}
{{- end }}
{{- if and .phaseswitching (ne .phaseswitching "false") }}
phaseswitching: {{ .phaseswitching }}
{{- end }}
{{- if and .meterinterval (ne .meterinterval "10s") }}
meterinterval: {{ .meterinterval }}
{{- end }}
{{- if ne .connecttimeout "5m" }}
connecttimeout: {{ .connecttimeout }}
{{- end }}
{{- end }}
