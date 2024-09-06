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
{{- if ne .remotestart "false"}}
remotestart: {{ .remotestart }}
{{- end }}
{{- if ne .connecttimeout "5m" }}
connecttimeout: {{ .connecttimeout }}
{{- end }}
{{- if and .timeout (ne .timeout "30s") }}
timeout: {{ .timeout }}
{{- end }}
{{- end }}
