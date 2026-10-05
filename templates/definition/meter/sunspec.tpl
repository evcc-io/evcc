{{ define "sunspec-maxacpower" }}
maxacpower: # nameplate rating
  source: sunspec
  {{- include "modbus" . | indent 2 }}
  value:
    - 120:WRtg
    - 702:WMaxRtg
{{- end }}

{{ define "sunspec-maxacpower-tcp" }}
maxacpower: # nameplate rating
  source: sunspec
  {{- include "modbus-connection" . | indent 2 }}
  id: 1
  value:
    - 120:WRtg
    - 702:WMaxRtg
{{- end }}
