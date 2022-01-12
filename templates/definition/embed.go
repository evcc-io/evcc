package definition

import "embed"

var (
	//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml
	YamlTemplates embed.FS

	//go:embed parambaselist.yaml
	ParamBaseListDefinition string

	//go:embed devicegroups.yaml
	DeviceGroupListDefinition string
)
