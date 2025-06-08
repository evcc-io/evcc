{{- define "param" }}
  {{ .Name }}:{{ if .Value }} {{ .Value }}{{ end }}
  {{- range .Values }}
  - {{ . }}
  {{- end }}
  {{- $unit := .Unit -}}
  {{- $description := localize .Description | replace "\n" " " | trim -}}
  {{- $help := localize .Help | replace "\n" " " | trim -}}
  {{- $choices := join ", " .Choice -}}
  {{- $optional := not .IsRequired -}}
  {{- if or $help $choices $optional $description }} # {{end}}
  {{- if $description }}{{ $description }}
    {{- if $unit }} ({{ $unit }}){{- end }}
    {{- if or $help $choices $optional }}, {{end}}
  {{- end}}
  {{- if $help }}{{ $help }} {{end}}
  {{- if $choices }}[{{ $choices }}] {{end}}
  {{- if $optional }}
    {{- if or $help $choices }}(optional){{ else }}optional{{end }}
  {{- end }}
{{- end }}

{{- define "header" }}
  type: template
  template: {{ .Template }}
  {{- if hasKey . "Usage" }}
  usage: {{ .Usage }}
  {{- end }}
{{- end }}

{{- define "default" }}
  {{- include "header" . }}
  {{- $usage := "" }}{{ if hasKey . "Usage" }}{{ $usage = .Usage }}{{ end }}
  {{- range .Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 }}
  {{- else if and (not .IsAdvanced) (or (not $usage) (not .Usages) (has $usage .Usages)) }}
  {{- template "param" . }}
  {{- end }}
  {{- end }}
{{- end }}

{{- define "advanced" }}
  {{- include "header" . }}
  {{- $usage := "" }}{{ if hasKey . "Usage" }}{{ $usage = .Usage }}{{ end }}
  {{- range .Params }}
  {{- if eq .Name "modbus" }}
  {{- $.Modbus | indent 2 }}
  {{- else if or (not $usage) (not .Usages) (has $usage .Usages) }}
  {{- template "param" . }}
  {{- end }}
  {{- end }}
{{- end -}}

template: {{ .Template }}
product:
  identifier: {{ .ProductIdentifier }}
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
{{- if .Countries }}
countries: ["{{ join "\", \"" .Countries }}"]
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
params:
  {{- range .Params }}
  {{- if and (not (eq .Name "usage")) (not .IsDeprecated) }}
  - name: {{ .Name | quote }}
    example: {{ .Example | quote }}
    default: {{ .Default }}
    choice: [{{ join ", " .Choice }}]
    unit: {{ .Unit }}
    {{- $description := localize .Description | replace "\n" " " | trim }}
    description: {{ $description | quote }}
    {{- $help := localize .Help | replace "\n" " " | trim }}
    help: {{ $help | quote }}
    advanced: {{ .IsAdvanced }}
    optional: {{ not .IsRequired }}
  {{- end }}
  {{- end }}
{{- if .ModbusData }}
modbus:
{{- range $key, $value := .ModbusData }}
  {{ $key }}: {{ $value }}
{{- end }}
{{- end }}