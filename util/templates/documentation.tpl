product:
{{- if .ProductBrand }}
  brand: {{ .ProductBrand }}
{{- end }}
{{- if .ProductDescription }}
  description: {{ .ProductDescription }}
{{- end }}
{{- if .ProductGroup }}
  group: {{ .ProductGroup }}
{{- end }}
{{- if .Capabilities }}
capabilities: ["{{ join "\", \"" .Capabilities }}"]
{{- end }}
{{- if .Requirements }}
requirements: ["{{ join "\", \"" .Requirements }}"]
{{- end }}
{{- if .RequirementDescription }}
description: |
{{ .RequirementDescription | indent 2 }}
{{- end }}
render:
{{- if .Usages -}}{{ range .Usages }}
  - usage: {{ . }}
    default: |
      type: template
      template: {{ $.Template }}
      usage: {{ . }}
      {{- range $.Params }}
      {{- if eq .Name "modbus" -}}
{{ $.Modbus | indent 6 -}}
      {{- else if ne .IsAdvanced true }}
      {{ .Name }}:
      {{- if len .Value }} {{ .Value }} {{- end }}
      {{- if ne (len .Values) 0 }}
      {{ range .Values }}
        - {{ . }}
      {{ end }}
      {{- end }}
      {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if eq .IsRequired false }} # Optional{{ end }}
      {{- end -}}
      {{- end -}}
{{- if $.AdvancedParams }}
    advanced: |
      type: template
      template: {{ $.Template }}
      usage: {{ . }}
      {{- range $.Params }}
      {{- if eq .Name "modbus" -}}
{{ $.Modbus | indent 6 -}}
      {{- else }}
      {{ .Name }}:
      {{- if len .Value }} {{ .Value }} {{- end }}
      {{- if ne (len .Values) 0 }}
      {{ range .Values }}
        - {{ . }}
      {{- end }}
      {{- end }}
      {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if ne .IsRequired true }} # Optional{{ end }}
      {{- end -}}
      {{- end -}}
{{ end }}
{{- end }}
{{- else }}
  - default: |
      type: template
      template: {{ $.Template }}
      {{- range $.Params }}
      {{- if eq .Name "modbus" -}}
{{ $.Modbus | indent 6 -}}
      {{- else if ne .IsAdvanced true }}
      {{ .Name }}:
      {{- if len .Value }} {{ .Value }} {{- end }}
      {{- if ne (len .Values) 0 }}
      {{ range .Values }}
        - {{ . }}
      {{- end }}
      {{- end }}
      {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if ne .IsRequired true }} # Optional{{ end }}
      {{- end -}}
      {{- end -}}
{{- if $.AdvancedParams }}
    advanced: |
      type: template
      template: {{ $.Template }}
      {{- range $.Params }}
      {{- if eq .Name "modbus" -}}
{{ $.Modbus | indent 6 -}}
      {{- else }}
      {{ .Name }}:
      {{- if len .Value }} {{ .Value }} {{- end }}
      {{- if ne (len .Values) 0 }}
      {{ range .Values }}
        - {{ . }}
      {{- end }}
      {{- end }}
      {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if ne .IsRequired true }} # Optional{{ end }}
      {{- end -}}
      {{- end -}}
{{- end }}
{{- end }}
