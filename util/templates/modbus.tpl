{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") (eq .modbus "rtuserial") }}
# Attached on local serial port (Modbus RTU)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- else if or (eq .modbus "rs485tcpip") (eq .modbus "rtutcp") }}
# Attached to transparent serial device gateway server (Modbus RTU over TCP/IP)
uri: {{ .host }}:{{ .port }}
rtu: true
{{- else if or (eq .modbus "tcpip") (eq .modbus "tcp") }}
# Modbus TCP (native interface or translated serial device)
uri: {{ .host }}:{{ .port }}
rtu: false
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- end }}
