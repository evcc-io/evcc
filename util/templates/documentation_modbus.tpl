{{- if .rs485serial }}

# Modbus: RS485 via adapter
modbus: rs485serial
id: {{ .id }}
device: {{ .device }} # USB-RS485 Adapter Adresse
baudrate: {{ .baudrate }} # Prüfe die Geräteeinstellungen, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: "{{ .comset }}" # Kommunikationsparameter für den Adapter
{{- end }}
{{- if .rs485tcpip }}

# Modbus: RS485 via TCPIP
modbus: rs485tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
rtu: true
{{- end }}
{{- if .tcpip }}

# Modbus: TCPIP
modbus: tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end }}
