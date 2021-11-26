uri: 0.0.0.0:7070 # uri for ui
interval: 10s # control cycle interval

log: info
{{- if ne (len .Meters) 0 }}

meters:
{{-   range .Meters }}
- {{ .Yaml | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Chargers) 0 }}

chargers:
{{-   range .Chargers }}
- {{ .Yaml | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Vehicles) 0 }}

vehicles:
{{-   range .Vehicles }}
- {{ .Yaml | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Chargers) 0 }}

loadpoints:
{{-   range .Loadpoints }}
- title: {{ .Title }}
  charger: {{ .Charger }}
{{-     if .ChargeMeter }}
  meters:
    charge: {{ .ChargeMeter }}
{{-     end }}
{{-     if ne (len .Vehicles) 0 }}
  vehicles:
{{-       range .Vehicles }}
  - {{ . }}
{{-       end }}
{{-     end }}
  mode: {{ .Mode }}
  phases: {{ .Phases }}
  mincurrent: {{ .MinCurrent }}
  maxcurrent: {{ .MaxCurrent }}
{{-   end }}
{{- end }}

site:
  title: {{ .Site.Title }}
  meters:
{{- if .Site.Grid }}
    grid: {{ .Site.Grid }}
{{- end }}
{{- if len .Site.PVs }}
    pvs:
{{-   range .Site.PVs }}
    - {{ . }}
{{-   end }}
{{- end }}
{{- if len .Site.Batteries }}
    batteries:
{{-   range .Site.Batteries }}
    - {{ . }}
{{-   end }}
{{- end }}
{{- if ne (len .EEBUS) 0 }}

eebus:
{{ .EEBUS | indent 2 }}
{{- end }}
{{- if ne (len .SponsorToken) 0 }}

sponsortoken: {{ .SponsorToken }}
{{- end}}
