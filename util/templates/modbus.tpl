{{- define "modbus" }}
id: {{ .id }}
{{- if or (eq .modbus "rs485serial") .rs485serial }}
# RS485 via adapter (Modbus RTU)
device: {{ .device }}
baudrate: {{ .baudrate }}
comset: {{ .comset }}
{{- else if or (eq .modbus "rs485tcpip") .rs485tcpip }}
# RS485 via TCP/IP (Modbus RTU)
uri: {{ joinHostPort .host .port }}
rtu: true
{{- else if or (eq .modbus "tcpip") .tcpip }}
# Modbus TCP
uri: {{ joinHostPort .host .port }}
rtu: false
{{- if .clientcert }}
# Modbus over TLS (mTLS)
clientcert: {{ .clientcert }}
clientkey: {{ .clientkey }}
{{- if .cacert }}
cacert: {{ .cacert }}
{{- end }}
{{- if .insecure }}
insecure: true
{{- end }}
{{- end }}
{{- else if or (eq .modbus "udp") .udp }}
# Modbus UDP
uri: {{ joinHostPort .host .port }}
udp: true
rtu: true
{{- else }}
# configuration error - should not happen
modbusConnectionTypeNotDefined: {{ .modbus }}
{{- end }}
{{- if .delay }}
delay: {{ .delay }}
{{- end }}
{{- if .timeout }}
timeout: {{ .timeout }}
{{- end }}
{{- end }}
