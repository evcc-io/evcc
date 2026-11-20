{{/* battery power limit: user value if configured, device maximum otherwise */}}
{{ define "sma-battery-limit" }}
{{- $value := .root.maxchargepower }}
{{- $read := 40189 }}
{{- $name := "BatConv.WMaxCha" }}
{{- if eq .limit "discharge" }}
{{- $value = .root.maxdischargepower }}
{{- $read = 40191 }}
{{- $name = "BatConv.WMaxDsch" }}
{{- end }}
{{- if $value }} {{ $value }} # W
{{- else }}
  source: modbus
  {{- include "modbus" .root | indent 2 }}
  register:
    address: {{ $read }} # SMA Modbus Profile: {{ $name }}
    type: holding
    decode: uint32
{{- end }}
{{- end }}

{{/* go setter body writing the battery power limit, negated if requested, to register .write */}}
{{ define "sma-battery-limit-set" }}
{{- $value := .root.maxchargepower }}
{{- $read := 40189 }}
{{- $name := "BatConv.WMaxCha" }}
{{- if eq .limit "discharge" }}
{{- $value = .root.maxdischargepower }}
{{- $read = 40191 }}
{{- $name = "BatConv.WMaxDsch" }}
{{- end }}
{{- if not $value }}
in:
- name: limit
  type: int
  config:
    source: modbus
    {{- include "modbus" .root | indent 4 }}
    register:
      address: {{ $read }} # SMA Modbus Profile: {{ $name }}
      type: holding
      decode: uint32
{{- end }}
script: |
  {{ if .negate }}-{{ end }}{{ if $value }}{{ $value }}{{ else }}limit{{ end }}
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
