{{- if .rs485serial }}

# RS485 via adapter (Modbus RTU)
modbus: rs485serial
id: {{ .id }}
device: {{ .device }} # USB-RS485 Adapter Adresse
baudrate: {{ .baudrate }} # Prüfe die Geräteeinstellungen, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: "{{ .comset }}" # Kommunikationsparameter für den Adapter
{{- if .delay }}
delay: {{ .delay }} # optional
{{- end }}
{{- if .timeout }}
timeout: {{ .timeout }} # optional
{{- end }}
{{- end }}
{{- if .rs485tcpip }}

# RS485 via TCP/IP (Modbus RTU)
modbus: rs485tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- if .delay }}
delay: {{ .delay }} # optional
{{- end }}
{{- if .timeout }}
timeout: {{ .timeout }} # optional
{{- end }}
{{- end }}
{{- if .tcpip }}

# Modbus TCP
modbus: tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- if .delay }}
delay: {{ .delay }} # optional
{{- end }}
{{- if .timeout }}
timeout: {{ .timeout }} # optional
{{- end }}
{{- end -}}
