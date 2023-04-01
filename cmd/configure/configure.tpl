# open evcc at http://evcc.local:7070
network:
  schema: http
  host: evcc.local # .local suffix announces the hostname on MDNS
  port: 7070

log: debug
levels:
  cache: error

# unique installation id
plant: {{ .Plant }}

interval: 10s # control cycle interval
{{- if .SponsorToken }}

sponsortoken: {{ .SponsorToken }}

# sponsors can set telemetry: true to enable anonymous data aggregation
# see https://github.com/evcc-io/evcc/discussions/4554
telemetry: {{ .Telemetry }}
{{- end}}
{{- if .Meters }}

meters:
{{- range .Meters }}
- {{ .Yaml | indent 2 | trim }}
{{- end }}
{{- end }}
{{- if .Chargers }}

chargers:
{{- range .Chargers }}
- {{ .Yaml | indent 2 | trim }}
{{- end }}
{{- end }}
{{- if .Vehicles }}

vehicles:
{{- range .Vehicles }}
- {{ .Yaml | indent 2 | trim }}
{{- end }}
{{- end }}
{{- if .Chargers }}

loadpoints:
{{- range .Loadpoints }}
- title: {{ .Title }}
  charger: {{ .Charger }}
{{- if .ChargeMeter }}
  meter: {{ .ChargeMeter }}
{{- end }}
{{- if .Vehicle }}
  vehicle: {{ .Vehicle }}
{{- end }}
  mode: {{ .Mode }}
  phases: {{ .Phases }}
  mincurrent: {{ .MinCurrent }}
  maxcurrent: {{ .MaxCurrent }}
  resetOnDisconnect: {{ .ResetOnDisconnect }}
{{- end }}
{{- end }}

site:
  title: {{ .Site.Title }}
  meters:
{{- if .Site.Grid }}
    grid: {{ .Site.Grid }}
{{- end }}
{{- if .Site.PVs }}
    pv:
    {{- range .Site.PVs }}
    - {{ . }}
    {{- end }}
{{- end }}
{{- if .Site.Batteries }}
    battery:
    {{- range .Site.Batteries }}
    - {{ . }}
    {{- end }}
{{- end }}
{{- if .Hems }}

hems:
{{ .Hems | indent 2 }}
{{- end }}
{{- if .EEBUS }}

eebus:
{{ .EEBUS | indent 2 }}
{{- end }}
