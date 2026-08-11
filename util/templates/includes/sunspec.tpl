{{ define "sunspec-maxacpower" }}
{{- if gt (float64 .maxacpower) 0.0 }}
maxacpower: {{ .maxacpower }} # W
{{- else }}
maxacpower: # nameplate rating
  source: sunspec
  {{- include "modbus" . | indent 2 }}
  value:
    - 120:WRtg
    - 702:WMaxRtg
{{- end }}
{{- end }}

{{ define "sunspec-maxacpower-uri" }}
{{- if gt (float64 .maxacpower) 0.0 }}
maxacpower: {{ .maxacpower }} # W
{{- else }}
maxacpower: # nameplate rating
  source: sunspec
  uri: {{ joinHostPort .host .port }}
  id: 1
  value:
    - 120:WRtg
    - 702:WMaxRtg
{{- end }}
{{- end }}
