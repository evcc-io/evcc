{{- if .rtuserial }}

# Attached to local serial interface (Modbus RTU)
modbus: rtu
id: {{ .id }}
device: {{ .device }} # Geräteadresse, typische Werte sind /dev/ttyUSB0, /dev/ttyAMA0, /dev/ttyS0 oder COM3
baudrate: {{ .baudrate }} # Baudrate, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: {{ .comset }} # Parität, Datenbits, Stoppbits, typische Werte sind 8N1, 8E1, 8O1
{{- end }}
{{- if .rtutcp }}

# Attached to transparent serial device server (Modbus RTU over TCP/IP)
modbus: rtu-over-tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end }}
{{- if .asciiserial }}

# Attached to local serial interface (Modbus ASCII)
modbus: ascii
id: {{ .id }}
device: {{ .device }} # Geräteadresse, typische Werte sind /dev/ttyUSB0, /dev/ttyAMA0, /dev/ttyS0 oder COM3
baudrate: {{ .baudrate }} # Baudrate, typische Werte sind 9600, 19200, 38400, 57600, 115200
comset: {{ .comset }} # Parität, Datenbits, Stoppbits, typische Werte sind 8N1, 8E1, 8O1
{{- end }}
{{- if .asciitcp }}

# Attached to transparent serial device server (Modbus ASCII over TCP/IP)
modbus: ascii-over-tcpip
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end }}
{{- if .tcp }}

# Modbus TCP
modbus: tcp
id: {{ .id }}
host: {{ .host }} # Hostname
port: {{ .port }} # Port
{{- end -}}
