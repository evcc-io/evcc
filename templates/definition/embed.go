package definition

import "embed"

// class folders contain device definitions (*.yaml) and include-only templates (*.tpl)
//
//go:embed charger meter vehicle tariff messenger circuit hems
var YamlTemplates embed.FS
