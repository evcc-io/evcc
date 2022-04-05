{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") .rs485serial }}
# RS485 via adapter (Modbus RTU)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- end }}
{{- if or (eq .modbus "rs485tcpip") .rs485tcpip }}
# RS485 via TCP/IP (Modbus RTU)
uri: {{ .host }}:{{ .port }}
rtu: true
{{- end }}
{{- if or (eq .modbus "tcpip") .tcpip }}
# Modbus TCP
uri: {{ .host }}:{{ .port }}
{{- end }}
{{- end}}
