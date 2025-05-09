{{- if .rs485serial }}

# RS485 via adapter (Modbus RTU)
modbus: rs485serial
id: {{ .id }}
device: {{ .device }} # USB-RS485 Adapter Adresse
baudrate: {{ .baudrate }} # Prüfe die Geräteeinstellungen, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: "{{ .comset }}" # Kommunikationsparameter für den Adapter
{{- end }}
{{- if .rs485tcpip }}

# RS485 via TCP/IP (Modbus RTU)
{{- if  eq .ProductBrand "Deye" }}
# Some Solarman data loggers support running as Modbus bridge. To connect to them use [pysolarman](https://github.com/jmccrohan/pysolarmanv5) in proxy mode
{{- end }}
modbus: rs485tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end }}
{{- if .tcpip }}

# Modbus TCP
modbus: tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end -}}
