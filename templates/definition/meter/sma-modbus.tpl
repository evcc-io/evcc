{{ define "sma-maxacpower" }}
maxacpower: # nameplate rating
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 30231 # SMA Modbus Profile: Inverter.WLim
    type: holding
    decode: uint32nan
{{- end }}

{{/* battery params, power limits fall back to the device maximum */}}
{{ define "sma-battery-params" }}
{{- include "battery-capacity" . }}
{{- include "battery-minmaxsoc" . }}
{{- if gt (.maxchargepower | int) 0 }}
maxchargepower: {{ .maxchargepower }} # W
{{- else }}
maxchargepower:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 40189 # SMA Modbus Profile: BatConv.WMaxCha
    type: holding
    decode: uint32
{{- end }}
{{- if gt (.maxdischargepower | int) 0 }}
maxdischargepower: {{ .maxdischargepower }} # W
{{- else }}
maxdischargepower:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 40191 # SMA Modbus Profile: BatConv.WMaxDsch
    type: holding
    decode: uint32
{{- end }}
{{- end }}

{{/* go setter writing the charge or discharge limit, optionally negated, to register .write */}}
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
    source: modbus
    {{- include "modbus" .root | indent 4 }}
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
