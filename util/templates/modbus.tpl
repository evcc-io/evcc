{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") (eq .modbus "rtuserial") (eq .modbus "asciiserial") }}
# Serial interface (Modbus RTU/ASCII)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- else if or (eq .modbus "rs485tcpip") (eq .modbus "rtutcp") }}
# Modbus RTU via TCP/IP
uri: {{ .host }}:{{ .port }}
rtu: true
{{- else if or (eq .modbus "asciitcp") }}
# Modbus ASCII via TCP/IP
uri: {{ .host }}:{{ .port }}
{{- else if or (eq .modbus "tcpip") (eq .modbus "tcp") }}
# Modbus TCP
uri: {{ .host }}:{{ .port }}
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- end }}