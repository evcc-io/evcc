{{/* battery params, power limits fall back to the device maximum */}}
{{ define "sma-battery-params" }}
{{- include "battery-capacity" . }}
{{- include "battery-minmaxsoc" . }}
maxchargepower:
  source: fallback
  value: {{ .maxchargepower | default "" }}
  fallback:
    source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 40189 # SMA Modbus Profile: BatConv.WMaxCha
      type: holding
      decode: uint32
maxdischargepower:
  source: fallback
  value: {{ .maxdischargepower | default "" }}
  fallback:
    source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 40191 # SMA Modbus Profile: BatConv.WMaxDsch
      type: holding
      decode: uint32
{{- end }}

{{/* fallback setter writing the configured power limit, or the device maximum, to register .write */}}
{{ define "sma-battery-limit" }}
{{- $value := .root.maxchargepower }}
{{- $read := 40189 }}
{{- $readname := "BatConv.WMaxCha" }}
{{- if eq .limit "discharge" }}
{{- $value = .root.maxdischargepower }}
{{- $read = 40191 }}
{{- $readname = "BatConv.WMaxDsch" }}
{{- end }}
value: {{ $value | default "" }}
fallback:
  source: modbus
  {{- include "modbus" .root | indent 2 }}
  register:
    address: {{ $read }} # SMA Modbus Profile: {{ $readname }}
    type: holding
    decode: uint32
{{- if .negate }}
scale: -1
{{- end }}
set:
  source: modbus
  {{- include "modbus" .root | indent 2 }}
  register:
    address: {{ .write }} # SMA Modbus Profile: {{ .name }}
    type: writemultiple
    decode: {{ if .negate }}int32{{ else }}uint32{{ end }}
{{- end }}
