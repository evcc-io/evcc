{{- define "modbus"}}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") .rs485serial }}
# RS485 via adapter:
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- end }}
{{- if or (eq .modbus "rs485tcpip") .rs485tcpip }}
# RS485 via TCPIP:
uri: {{ .host }}:{{ .port }}
rtu: true # serial modbus rtu (rs485) device connected using simple ethernet adapter
{{- end }}
{{- if or (eq .modbus "tcpip") .tcpip }}
# TCPIP
uri: {{ .host }}:{{ .port }}
{{- end }}
{{- end}}
