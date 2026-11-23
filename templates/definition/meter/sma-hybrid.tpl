{{/* hybrid inverter grid meter */}}
{{ define "sma-hybrid-grid" }}
power:
  source: calc
  add:
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30865 # SMA Modbus Profile: Metering.GridMs.TotWIn
        type: input
        decode: int32nan
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30867 # SMA Modbus Profile: Metering.GridMs.TotWOut
        type: input
        decode: int32nan
      scale: -1
energy:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 30581 # SMA Modbus Profile: Metering.GridMs.TotWhIn
    type: holding
    decode: int32nan
  scale: 0.001
currents:
  - source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 31435 # SMA Modbus Profile: Metering.GridMs.A.phsA # Grid current phase L1, unsigned (independent of flow direction)
      type: input
      decode: int32nan
    scale: 0.001
  - source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 31437 # SMA Modbus Profile: Metering.GridMs.A.phsB # Grid current phase L2, unsigned (independent of flow direction)
      type: input
      decode: int32nan
    scale: 0.001
  - source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 31439 # SMA Modbus Profile: Metering.GridMs.A.phsC # Grid current phase L3, unsigned (independent of flow direction)
      type: input
      decode: int32nan
    scale: 0.001
{{- end }}

{{/* grid draw per phase, missing on Sunny Boy Smart Energy */}}
{{ define "sma-hybrid-grid-powers" }}
powers:
  - source: calc
    add:
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31265 # SMA Modbus Profile: Metering.GridMs.WIn.phsA # Power drawn from grid phase L1, unsigned, zero if no consumption
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: 1
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31259 # SMA Modbus Profile: Metering.GridMs.W.phsA # Power grid feeding L1, unsigned, zero if no feeding
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: -1
  - source: calc
    add:
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31267 # SMA Modbus Profile: Metering.GridMs.WIn.phsB # Power drawn from grid phase L2, unsigned, zero if no consumption
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: 1
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31261 # SMA Modbus Profile: Metering.GridMs.W.phsB # Power grid feeding L2, unsigned, zero if no feeding
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: -1
  - source: calc
    add:
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31269 # SMA Modbus Profile: Metering.GridMs.WIn.phsC # Power drawn from grid phase L3, unsigned, zero if no consumption
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: 1
      - source: modbus
        {{- include "modbus" . | indent 8 }}
        register:
          address: 31263 # SMA Modbus Profile: Metering.GridMs.W.phsC # Power grid feeding L3, unsigned, zero if no feeding
          type: input
          decode: uint32nan
        block: # 31259-31270
          register: 31259
          count: 12
        scale: -1
{{- end }}

{{/* hybrid inverter pv dc power */}}
{{ define "sma-hybrid-pv" }}
power:
  source: calc
  add:
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30773 # SMA Modbus Profile: DcMs.Watt [0]
        type: holding
        decode: int32nan
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30961 # SMA Modbus Profile: DcMs.Watt [1]
        type: holding
        decode: int32nan
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30967 # SMA Modbus Profile: DcMs.Watt [2]
        type: holding
        decode: int32nan
maxacpower: {{ .maxacpower }} # W
{{- end }}

{{/* hybrid inverter battery power and soc */}}
{{ define "sma-hybrid-battery" }}
power:
  source: calc
  mul:
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30843 # SMA Modbus Profile: Bat.Amp, positive while discharging
        type: holding
        decode: int32nan
      scale: 0.001
    - source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 30851 # SMA Modbus Profile: Bat.Vol
        type: holding
        decode: uint32nan
      scale: 0.01
# no energy: BMS counters BatDsch.ActBatDsch (34665) and BatChrg.ActBatChrg (34661) only count whole kWh, power is integrated instead
soc:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 30845 # SMA Modbus Profile: Bat.ChaStt
    type: holding
    decode: uint32nan
{{- end }}
