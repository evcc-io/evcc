package definition

import "embed"

//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml tariff/*.yaml messenger/*.yaml circuit/*.yaml hems/*.yaml
var YamlTemplates embed.FS

//go:embed */*.tpl
var TemplateIncludes embed.FS
