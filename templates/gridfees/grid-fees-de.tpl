template: grid-fees-de
products:
  - description:
      de: Netzentgelte §14a EnWG (Modul 3)
      en: Grid fees §14a EnWG (module 3)
requirements:
  description:
    de: |
      Zeitvariable Netzentgelte nach §14a EnWG Modul 3, Netto-Preise der Preisblätter [[ .Year ]].
      Datenquelle: [github.com/ScumbagSteve/Grid-fees](https://github.com/ScumbagSteve/Grid-fees).
    en: |
      Time-variable grid fees according to §14a EnWG module 3, net prices of the [[ .Year ]] price sheets.
      Data source: [github.com/ScumbagSteve/Grid-fees](https://github.com/ScumbagSteve/Grid-fees).
group: price
countries: ["DE"]
params:
  - name: gridoperator
    type: choice
    required: true
    default: [[ (index .Operators 0).Name ]]
    description:
      de: Netzbetreiber
      en: Grid operator
    choice:
[[- range .Operators ]]
      - [[ .Name ]]
[[- end ]]
  - preset: tariff-base
render: |
  type: fixed
[[- range $i, $o := .Operators ]]
  {{- [[ if $i ]]else [[ end ]]if eq .gridoperator "[[ $o.Name ]]" }}
  price: [[ $o.Price ]]
  zones:
  [[- range $o.Zones ]]
    - price: [[ .Price ]]
      hours: [[ .Hours ]]
    [[- if .Months ]]
      months: [[ .Months ]]
    [[- end ]]
  [[- end ]]
[[- end ]]
  {{- else }}
  {{ fail (printf "unknown grid operator: %s" .gridoperator) }}
  {{- end }}
  {{ include "tariff-base" . }}
