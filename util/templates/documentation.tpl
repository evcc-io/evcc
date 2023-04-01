{{- define "param" }}
  {{ .Name }}:
  {{- if .Value }} {{ .Value }} {{- end }}
  {{- range .Values }}
  - {{ . }}
  {{- end }}
  {{- if .Help.DE }} # {{ .Help.DE }}{{- end }}{{- if not .IsRequired }} # Optional{{- end }}
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
  {{- $usage := "" }}{{ if hasKey . "Usage" }}{{ $usage = .Usage }}{{ end }}
  {{- range $.Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 }}
  {{- else if and (not .IsAdvanced) (or (or (not $usage) (not .Usages)) (and $usage .Usages (has $usage .Usages))) }}
  {{- template "param" . }}
  {{- end }}
  {{- end }}
{{- end }}

{{- define "advanced" }}
  {{- include "header" . }}
  {{- $usage := "" }}{{ if hasKey . "Usage" }}{{ $usage = .Usage }}{{ end }}
  {{- range $.Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 }}
  {{- else if or (or (not $usage) (not .Usages)) (and $usage .Usages (has $usage .Usages)) }}
  {{- template "param" . }}
  {{- end }}
  {{- end }}
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
