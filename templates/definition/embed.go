package definition

import "embed"

var (
	//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml
	YamlTemplates embed.FS
)
