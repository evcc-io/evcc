{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") .rs485serial }}
# RS485 via adapter (Modbus RTU)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: {{ .comset }}
{{- else if or (eq .modbus "rs485tcpip") .rs485tcpip }}
# RS485 via TCP/IP (Modbus RTU)
uri: {{ .host }}:{{ .port }}
rtu: true
{{- else if or (eq .modbus "tcpip") .tcpip }}
# Modbus TCP
uri: {{ .host }}:{{ .port }}
rtu: false
{{- else if or (eq .modbus "udp") .udp }}
# Modbus UDP
uri: {{ .host }}:{{ if (ne .port "502") }}{{ .port }}{{ else }}8899{{ end }}
udp: true
rtu: true
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- end }}
