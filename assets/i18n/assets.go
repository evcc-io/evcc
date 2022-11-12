package i18n

import (
	"embed"
)

//go:embed *.toml
var LocaleFS embed.FS
