{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") (eq .modbus "rtu") (eq .modbus "ascii") }}
# Local serial interface (Modbus RTU/ASCII)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- else if or (eq .modbus "rs485tcpip") (eq .modbus "rtu-over-tcpip") }}
# Modbus RTU over TCP/IP
uri: {{ .host }}:{{ .port }}
rtu: true
{{- else if or (eq .modbus "ascii-over-tcpip") }}
# Modbus ASCII over TCP/IP
uri: {{ .host }}:{{ .port }}
{{- else if or (eq .modbus "tcpip") (eq .modbus "tcp") }}
# Modbus TCP
uri: {{ .host }}:{{ .port }}
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- end }}