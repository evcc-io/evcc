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

{{/* battery control via BMS operating mode and power limits */}}
{{ define "sma-batterymode" }}
batterymode:
  source: watchdog
  timeout: {{ .watchdog }}
  reset: 1 # reset watchdog on normal
  set:
    source: switch
    switch:
    - case: 1 # normal
      set:
        source: sequence
        set:
        - source: const
          value: 2424 # Voreinstellung (Dft)
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40236 # SMA Modbus Profile: CmpBMS.OpMod - Betriebsart des BMS
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40793 # SMA Modbus Profile: CmpBMS.BatChaMinW - Minimale Batterieladeleistung
              type: writemultiple
              decode: uint32
        - source: go # charge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "charge" "write" 40795 "name" "CmpBMS.BatChaMaxW - Maximale Batterieladeleistung") | indent 10 }}
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40797 # SMA Modbus Profile: CmpBMS.BatDschMinW - Minimale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: go # discharge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "discharge" "write" 40799 "name" "CmpBMS.BatDschMaxW - Maximale Batterieentladeleistung") | indent 10 }}
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40801 # SMA Modbus Profile: CmpBMS.GridWSpt - Sollwert der Netzaustauschleistung
              type: writemultiple
              decode: uint32
    - case: 2 # holddischarge
      set:
        source: sequence
        set:
        - source: const
          value: 2424 # Voreinstellung (Dft)
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40236 # SMA Modbus Profile: CmpBMS.OpMod - Betriebsart des BMS
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40793 # SMA Modbus Profile: CmpBMS.BatChaMinW - Minimale Batterieladeleistung
              type: writemultiple
              decode: uint32
        - source: go # charge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "charge" "write" 40795 "name" "CmpBMS.BatChaMaxW - Maximale Batterieladeleistung") | indent 10 }}
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40797 # SMA Modbus Profile: CmpBMS.BatDschMinW - Minimale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40799 # SMA Modbus Profile: CmpBMS.BatDschMaxW - Maximale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40801 # SMA Modbus Profile: CmpBMS.GridWSpt - Sollwert der Netzaustauschleistung
              type: writemultiple
              decode: uint32
    - case: 3 # charge
      set:
        source: sequence
        set:
        - source: const
          value: 2289 # Batterie laden (BatChaMod)
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40236 # SMA Modbus Profile: CmpBMS.OpMod - Betriebsart des BMS
              type: writemultiple
              decode: uint32
        - source: go # charge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "charge" "write" 40793 "name" "CmpBMS.BatChaMinW - Minimale Batterieladeleistung") | indent 10 }}
        - source: go # charge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "charge" "write" 40795 "name" "CmpBMS.BatChaMaxW - Maximale Batterieladeleistung") | indent 10 }}
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40797 # SMA Modbus Profile: CmpBMS.BatDschMinW - Minimale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40799 # SMA Modbus Profile: CmpBMS.BatDschMaxW - Maximale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40801 # SMA Modbus Profile: CmpBMS.GridWSpt - Sollwert der Netzaustauschleistung
              type: writemultiple
              decode: uint32
    - case: 4 # holdcharge
      set:
        source: sequence
        set:
        - source: const
          value: 2424 # Voreinstellung (Dft)
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40236 # SMA Modbus Profile: CmpBMS.OpMod - Betriebsart des BMS
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40793 # SMA Modbus Profile: CmpBMS.BatChaMinW - Minimale Batterieladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40795 # SMA Modbus Profile: CmpBMS.BatChaMaxW - Maximale Batterieladeleistung
              type: writemultiple
              decode: uint32
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40797 # SMA Modbus Profile: CmpBMS.BatDschMinW - Minimale Batterieentladeleistung
              type: writemultiple
              decode: uint32
        - source: go # discharge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "discharge" "write" 40799 "name" "CmpBMS.BatDschMaxW - Maximale Batterieentladeleistung") | indent 10 }}
        - source: const
          value: 0
          set:
            source: modbus
            {{- include "modbus" . | indent 12 }}
            register:
              address: 40801 # SMA Modbus Profile: CmpBMS.GridWSpt - Sollwert der Netzaustauschleistung
              type: writemultiple
              decode: uint32
{{- end }}
