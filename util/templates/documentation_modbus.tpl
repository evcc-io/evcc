{{- if .rtuserial }}

# Attached to local serial interface (Modbus RTU)
modbus: rtu # modbus protocol
id: {{ .id }} # modbus id
device: {{ .device }} # serial device name e.g. /dev/ttyUSB0, /dev/ttyAMA0, /dev/ttyS0 or COM3 (on Windows)
baudrate: {{ .baudrate }} # baud rate e.g. 9600, 19200, 38400, 57600, 115200
comset: {{ .comset }} # data bits, parity mode (none, even, odd), stop bits e.g. 8n1, 8e1 or 8o1
{{- end }}
{{- if .rtutcp }}

# Attached to transparent serial device server (Modbus RTU over TCP/IP)
modbus: rtu-over-tcpip # modbus protocol
id: {{ .id }} # modbus id
host: {{ .host }} # hostname / IP address
port: {{ .port }} # tcp port
{{- end }}
{{- if .asciiserial }}

# Attached to local serial interface (Modbus ASCII)
modbus: ascii # modbus protocol
id: {{ .id }} # modbus id
device: {{ .device }} # serial device name e.g. /dev/ttyUSB0, /dev/ttyAMA0, /dev/ttyS0 or COM3 (on Windows)
baudrate: {{ .baudrate }} # baud rate e.g. 9600, 19200, 38400, 57600, 115200
comset: {{ .comset }} # data bits, parity mode (none, even, odd), stop bits e.g. 8n1, 8e1 or 8o1
{{- end }}
{{- if .asciitcp }}

# Attached to transparent serial device server (Modbus ASCII over TCP/IP)
modbus: ascii-over-tcpip # modbus protocol
id: {{ .id }} # modbus id
host: {{ .host }} # hostname / IP address
port: {{ .port }} # port
{{- end }}
{{- if .tcp }}

# Modbus TCP
modbus: tcp # modbus protocol
id: {{ .id }} # modbus id
host: {{ .host }} # hostname / IP address
port: {{ .port }} # port
{{- end -}}
