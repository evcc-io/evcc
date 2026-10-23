{{- define "solarassistant-meter" }}
{{- if eq .usage "grid" }}
power:
  source: mqtt
  {{- include "mqtt" . | indent 2 }}
  topic: {{ .topic }}/grid_power/state
{{- end }}
{{- if eq .usage "pv" }}
power:
  {{- if .excludeaux }}
  source: calc
  add:
  - source: mqtt
    {{- include "mqtt" . | indent 4 }}
    topic: {{ .topic }}/pv_power/state
  - source: mqtt
    {{- include "mqtt" . | indent 4 }}
    topic: {{ .topic }}/auxiliary_pv_power/state
    scale: -1
  {{- else }}
  source: mqtt
  {{- include "mqtt" . | indent 2 }}
  topic: {{ .topic }}/pv_power/state
  {{- end }}
{{- end }}
{{- if eq .usage "battery" }}
power:
  source: mqtt
  {{- include "mqtt" . | indent 2 }}
  topic: {{ .topic }}/battery_power/state
  scale: -1 # SolarAssistant: negative = discharging
soc: # published on change only, so no staleness timeout
  source: mqtt
  {{- include "mqtt" (omit . "timeout") | indent 2 }}
  topic: {{ .topic }}/battery_state_of_charge/state
{{- include "battery-params" . }}
{{- end }}
{{- end }}

{{- define "solarassistant-set" }}
- source: const
  value: {{ .value }}
  set:
    source: mqtt
    {{- include "mqtt" .root | indent 4 }}
    topic: {{ .root.topic }}/{{ .name }}/set
    {{- if .payload }}
    payload: {{ quote .payload }}
    {{- end }}
{{- if .root.writedelay }}
- source: sleep
  duration: {{ .root.writedelay }}
{{- end }}
{{- end }}

{{- define "solarassistant-peakshaving" }}
{{- if .gridpeakshavingpower }}
{{- include "solarassistant-set" (dict "root" . "name" "grid_peak_shaving_power" "value" .gridpeakshavingpower) }}
{{- include "solarassistant-set" (dict "root" . "name" "grid_peak_shaving" "value" 0 "payload" "Enabled") }}
{{- end }}
{{- end }}

{{- define "solarassistant-batterymode" }}
# Deye/Sunsynk 3-phase LV only. Requires all six TOU time points at 00:00 (program 6 then covers the whole day),
# "Use timer" on and "Grid charge" enabled. Restrictive writes come first, so a failed write leaves a safe state.
# The program 6 grid charge switch takes 1 and 0.
batterymode:
  source: switch
  switch:
  - case: 1 # normal: self-consumption down to minsoc
    set:
      source: sequence
      set:
      {{- include "solarassistant-peakshaving" . | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "work_mode" "value" 0 "payload" .workmode) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "grid_charge_point_6" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_grid_charge_current" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "capacity_point_6" "value" .minsoc) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "power_point_6" "value" .maxdischargepower) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_charge_current" "value" .maxchargecurrent) | indent 6 }}
  - case: 2 # hold: no discharge, PV may still charge
    set:
      source: sequence
      set:
      {{- include "solarassistant-set" (dict "root" . "name" "power_point_6" "value" 0) | indent 6 }}
      {{- include "solarassistant-peakshaving" . | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "work_mode" "value" 0 "payload" .workmode) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "grid_charge_point_6" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_grid_charge_current" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "capacity_point_6" "value" .maxsoc) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_charge_current" "value" .maxchargecurrent) | indent 6 }}
  - case: 3 # charge: program 6 charges from grid up to maxsoc, PV is not throttled
    set:
      source: sequence
      set:
      # peak shaving caps the grid import, the battery takes the rest
      {{- include "solarassistant-peakshaving" . | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "work_mode" "value" 0 "payload" .workmode) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "capacity_point_6" "value" .maxsoc) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "power_point_6" "value" .maxdischargepower) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_charge_current" "value" .maxchargecurrent) | indent 6 }}
      # ceiling on the total battery charge current, the grid only tops up what PV does not deliver
      {{- include "solarassistant-set" (dict "root" . "name" "max_grid_charge_current" "value" .gridchargecurrent) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "grid_charge_point_6" "value" 1) | indent 6 }}
  - case: 4 # holdcharge: no charging, discharge to the house allowed
    set:
      source: sequence
      set:
      {{- include "solarassistant-set" (dict "root" . "name" "max_charge_current" "value" 0) | indent 6 }}
      {{- include "solarassistant-peakshaving" . | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "work_mode" "value" 0 "payload" .workmode) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "grid_charge_point_6" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_grid_charge_current" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "capacity_point_6" "value" .minsoc) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "power_point_6" "value" .maxdischargepower) | indent 6 }}
  - case: 5 # discharge: sell from the battery down to minsoc
    set:
      source: sequence
      set:
      {{- include "solarassistant-set" (dict "root" . "name" "grid_charge_point_6" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_grid_charge_current" "value" 0) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "max_charge_current" "value" .maxchargecurrent) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "capacity_point_6" "value" .minsoc) | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "power_point_6" "value" .maxdischargepower) | indent 6 }}
      # peak shaving blocks selling, Selling first allows battery export
      {{- include "solarassistant-set" (dict "root" . "name" "grid_peak_shaving" "value" 0 "payload" "Disabled") | indent 6 }}
      {{- include "solarassistant-set" (dict "root" . "name" "work_mode" "value" 0 "payload" "Selling first") | indent 6 }}
{{- end }}
