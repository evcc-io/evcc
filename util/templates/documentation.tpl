{{- define "param" }}
  {{ .Name }}:
  {{- if .Value }} {{ .Value }} {{- end }}
  {{- if .Values }}
  {{ range .Values }}
  - {{ . }}
  {{- end }}
  {{- end }}
  {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}
>{{ printf "%+v" . }}
  {{/* {{- if ne .IsRequired true }} # Optional{{ end }} */}}
{{- end }}

{{- define "header" }}
  type: template
  template: {{ $.Template }}
  {{- if hasKey . "Usage" }}
  usage: {{ .Usage }}
  {{- end }}
{{- end }}

{{- define "default" }}
  {{- include "header" . }}
  {{- range $.Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 -}}
  {{- else if ne .IsAdvanced true }}
  {{ .Name }}:
  {{- if .Value }} {{ .Value }} {{- end }}
  {{- if .Values }}
  {{ range .Values }}
  - {{ . }}
  {{- end }}
  {{- end }}
  {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if ne .IsRequired true }} # Optional{{ end }}
  {{- end -}}
  {{- end -}}
{{- end }}

{{- define "advanced" }}
  {{- include "header" . }}
  {{- range $.Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 -}}
  {{- else }}
  {{ .Name }}:
  {{- if .Value }} {{ .Value }} {{- end }}
  {{- if .Values }}
  {{ range .Values }}
  - {{ . }}
  {{- end }}
  {{- end }}
  {{- if .Help.DE }} # {{ .Help.DE }}{{ end }}{{ if ne .IsRequired true }} # Optional{{ end }}
  {{- end -}}
  {{- end -}}
{{- end -}}

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
{{- if .Usages -}}
{{- $content := . }}
{{- range $usage := .Usages }}
{{- $_ := set $content "Usage" $usage }}
  - usage: {{ $usage }}
    default: |
	{{- include "default" $content | indent 4 }}
    {{- if $.AdvancedParams }}
    advanced: |
	{{- include "advanced" $content | indent 4 }}
    {{- end }}
{{- end }}
{{- else }}
  - default: |
    {{- include "default" . | indent 4 }}
    {{- if $.AdvancedParams }}
    advanced: |
    {{- include "advanced" . | indent 4 }}
    {{- end }}
{{- end }}
