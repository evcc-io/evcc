package definition

import "embed"

// directories don't embed files starting with _ like the generated _required.yaml
//
//go:embed charger meter vehicle tariff messenger circuit hems curtailer
var YamlTemplates embed.FS

//go:embed */*.tpl
var TemplateIncludes embed.FS
