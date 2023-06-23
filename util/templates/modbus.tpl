{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") (eq .modbus "serial") }}
# RS485 via USB adapter (Modbus RTU)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- else if or (eq .modbus "rs485tcpip") (eq .modbus "rtu") }}
# RS485 via Ethernet adapter (Modbus RTU)
uri: {{ .host }}:{{ .port }}
rtu: true
{{- else if or (eq .modbus "tcpip") (eq .modbus "tcp") }}
# Modbus TCP
uri: {{ .host }}:{{ .port }}
rtu: false
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- end }}
