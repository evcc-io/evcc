{{- define "param" }}
  {{ .Name }}:{{ if .Value }} {{ .Value }}{{ end }}
  {{- range .Values }}
  - {{ . }}
  {{- end }}
  {{- $help := localize .Help | replace "\n" " " -}}
  {{- $choice := .Choice }}

  {{- if or (not .IsRequired) $help $choice }} #
    {{- if not .IsRequired }} optional{{- end }}
    {{- if $help }}
      {{- if not .IsRequired }};{{- end }} {{ $help }}
    {{- end }}
    {{- if $choice }}
      {{- if or (not .IsRequired) $help }};{{- end }}
      {{- if kindIs "string" $choice }}
        {{- if regexMatch "^[0-9]+$" $choice }} Choices: {{ $choice }} {{- else }} Choices: "{{ $choice }}" {{- end }}
      {{- else if kindIs "slice" $choice }} Choices: {{ range $index, $element := $choice }}{{ if $index }}, {{ end }}{{- if kindIs "string" $element }}
          {{- if regexMatch "^[0-9]+$" $element }}{{ $element }}{{- else }}"{{ $element }}"{{ end }}
        {{- else }}{{ $element }}{{- end }}{{ end }}
      {{- end }}
    {{- end }}
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
