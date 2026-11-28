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
        - source: fallback # charge limit
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
        - source: fallback # discharge limit
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
        - source: fallback # charge limit
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
        - source: fallback # charge limit
          {{- include "sma-battery-limit" (dict "root" . "limit" "charge" "write" 40793 "name" "CmpBMS.BatChaMinW - Minimale Batterieladeleistung") | indent 10 }}
        - source: fallback # charge limit
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
        - source: fallback # discharge limit
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
