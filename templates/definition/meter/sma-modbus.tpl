{{/* go setter writing the configured power limit, or the device maximum read once, to register .write */}}
{{ define "sma-battery-limit" }}
{{- $value := .root.maxchargepower }}
{{- $read := 40189 }}
{{- $readname := "BatConv.WMaxCha" }}
{{- if eq .limit "discharge" }}
{{- $value = .root.maxdischargepower }}
{{- $read = 40191 }}
{{- $readname = "BatConv.WMaxDsch" }}
{{- end }}
in:
- name: limit
  type: int
  config:
    {{- if gt ($value | int) 0 }}
    source: const
    value: {{ $value }}
    {{- else }}
    source: cached
    cache: 720h # nameplate rating
    value:
      source: modbus
      {{- include "modbus" .root | indent 6 }}
      register:
        address: {{ $read }} # SMA Modbus Profile: {{ $readname }}
        type: holding
        decode: uint32
    {{- end }}
script: |
  {{ if .negate }}-{{ end }}limit
out:
- name: limit
  type: int
  config:
    source: modbus
    {{- include "modbus" .root | indent 4 }}
    register:
      address: {{ .write }} # SMA Modbus Profile: {{ .name }}
      type: writemultiple
      decode: {{ if .negate }}int32{{ else }}uint32{{ end }}
{{- end }}
