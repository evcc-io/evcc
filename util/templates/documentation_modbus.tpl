id: {{ .id }}
{{- if.modbusrs485serial }}
# RS485 via adapter:
device: {{ .device }} # USB-RS485 Adapter Adresse
baudrate: {{ .baudrate }} # Prüfe die Geräteeinstellungen, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: "{{ .comset }}" # Kommunikationsparameter für den Adapter
{{- end }}
{{- if .modbusrs485tcpip }}
# RS485 via TCPIP:
uri: {{ .host }}:{{ .port }} # IP-Adresse oder Hostname: Port
rtu: true
{{- end }}
{{- if .modbustcpip }}
# TCPIP
uri: {{ .host }}:{{ .port }} # IP-Adresse oder Hostname: Port
{{- end }}
