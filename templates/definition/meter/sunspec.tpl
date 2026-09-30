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
  uri: {{ joinHostPort .host .port }}
  id: 1
  value:
    - 120:WRtg
    - 702:WMaxRtg
{{- end }}
