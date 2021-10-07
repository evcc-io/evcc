log: info
{{- if ne (len .Meters) 0 }}

meters:
{{-   range .Meters }}
- {{ .Yaml  | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Chargers) 0 }}

chargers:
{{-   range .Chargers }}
- {{ .Yaml  | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Vehicles) 0 }}

vehicles:
{{-   range .Vehicles }}
- {{ .Yaml  | indent 2 | trim }}
{{-   end }}
{{- end }}
{{- if ne (len .Chargers) 0 }}

loadpoints:
{{-   range .Loadpoints }}
- title: {{ .Title }}
  charger: {{ .Charger }}
{{-     if .Meter }}
  meters:
    charge: {{ .Meter }}
{{-     end }}
{{-     if ne (len .Vehicles) 0 }}
  vehicles:
{{-       range .Vehicles }}
  - {{ . }}
{{-       end }}
{{-     end }}
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
{{- if .Site.Battery }}
    battery: {{ .Site.Battery }}
{{- end }}
{{- if ne (len .EEBUS) 0 }}

eebus:
{{ .EEBUS | indent 2 }}
{{- end }}
