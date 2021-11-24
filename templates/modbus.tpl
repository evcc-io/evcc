id: {{ .id }}
{{- if or (eq .modbus "rs485serial") .ModbusRS485Serial }}
# RS485 via adapter:
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: "{{ .comset }}"
{{- end }}
{{- if or (eq .modbus "rs485tcpip") .ModbusRS485TCPIP }}
# RS485 via TCPIP:
uri: {{ .host }}:{{ .port }}
rtu: true # serial modbus rtu (rs485) device connected using simple ethernet adapter
{{- end }}
{{- if or (eq .modbus "tcpip") .ModbusTCPIP }}
# TCPIP
uri: {{ .host }}:{{ .port }}
{{- end }}