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
{{- if and .remotestart (ne .remotestart "false") }}
remotestart: {{ .remotestart }}
{{- end }}
{{- if .metervalues }}
metervalues: {{ .metervalues }}
{{- end }}
{{- if and .meterinterval (ne .meterinterval "10s") }}
meterinterval: {{ .meterinterval }}
{{- end }}
{{- if ne .connecttimeout "5m" }}
connecttimeout: {{ .connecttimeout }}
{{- end }}
{{- if and .timeout (ne .timeout "30s") }}
timeout: {{ .timeout }}
{{- end }}
{{- end }}
