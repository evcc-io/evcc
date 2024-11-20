package definition

import "embed"

//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml tariff/*.yaml
var YamlTemplates embed.FS
