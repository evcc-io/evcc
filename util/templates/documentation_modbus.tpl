{{- if .serial }}

# RS485 via USB adapter (Modbus RTU)
modbus: serial
id: {{ .id }}
device: {{ .device }} # USB Geräteadresse, typische Werte sind /dev/ttyUSB0, /dev/ttyAMA0, /dev/ttyS0
baudrate: {{ .baudrate }} # Baudrate, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: {{ .comset }} # Parität, Datenbits, Stoppbits, typische Werte sind 8N1, 8E1, 8O1
{{- end }}
{{- if .rtu }}

# RS485 via Ethernet adapter (Modbus RTU)
modbus: rtu
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end }}
{{- if .tcp }}

# RS485 via protocol converter (Modbus TCP)
modbus: tcp
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end -}}
{{- if .tcpip }}

# Modbus TCP
modbus: tcp
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end -}}
