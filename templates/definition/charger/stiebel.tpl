{{ define "stiebel-sgready" }}
type: sgready
getmode:
  source: map
  values:
    1: 1 # Frostschutz
    2: 2 # Normal
    3: 3 # Forcierter Betrieb
    4: 3 # Sofortige Ansteuerung
  get:
    source: modbus
    {{- include "modbus" . | indent 4 }}
    register:
      address: 5000
      type: input
      encoding: uint16
setmode:
  source: switch
  switch:
  - case: 2 # normal
    set:
      source: sequence
      set:
      - source: const
        value: 0
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4001
            type: writesingle
            encoding: uint16
      - source: const
        value: 0
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4002
            type: writesingle
            encoding: uint16
  - case: 3 # boost
    set:
      source: sequence
      set:
      - source: const
        value: 1
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4001
            type: writesingle
            encoding: uint16
      - source: const
        value: 0
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4002
            type: writesingle
            encoding: uint16
  - case: 1 # dimm
    set:
      source: sequence
      set:
      - source: const
        value: 0
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4001
            type: writesingle
            encoding: uint16
      - source: const
        value: 1
        set:
          source: modbus
          {{- include "modbus" . | indent 10 }}
          register:
            address: 4002
            type: writesingle
            encoding: uint16
{{- end }}

{{ define "stiebel-wpm-base" }}
{{- include "stiebel-sgready" . }}
{{- if .tempsource }}
temp:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: {{ if eq .tempsource "warmwater" -}} 521 {{ else }} 517 {{- end }}
    type: input
    encoding: int16
  scale: 0.1
limittemp:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: {{ if eq .tempsource "warmwater" -}} 522 {{ else }} 518 {{- end }} # 523 WW-Solltemperatur / 519 Puffer-Solltemperatur (1-based)
    type: input
    encoding: int16
  scale: 0.1
{{- end }}
{{- end }}

{{ define "stiebel-wpm-energy" }}
energy: # LA Verdichter Heizen + Warmwasser in kWh
  source: go
  in:
  # kWh and MWh parts are read as one uint32 per counter so a 999 -> 1000 kWh rollover cannot mix both sides
  - name: heating
    type: int
    config:
      source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 3511 # 3512 LA Heizen Summe kWh, 3513 MWh (1-based)
        type: input
        encoding: uint32s
  - name: dhw
    type: int
    config:
      source: modbus
      {{- include "modbus" . | indent 6 }}
      register:
        address: 3514 # 3515 LA Warmwasser Summe kWh, 3516 MWh (1-based)
        type: input
        encoding: uint32s
  script: |
    float64((heating>>16)*1000 + (heating & 0xffff) + (dhw>>16)*1000 + (dhw & 0xffff))
{{- end }}

{{ define "stiebel-wpm-power" }}
power:
  source: modbus
  {{- include "modbus" . | indent 2 }}
  register:
    address: 3679 # 3680 INVERTER AUFNAHMELEISTUNG IWS 1 (1-based)
    type: input
    encoding: int16nan
  scale: 100 # 0.1 kW -> W
{{- end }}
