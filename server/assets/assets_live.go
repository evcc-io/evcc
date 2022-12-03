//go:build !release

package assets

import (
	"os"
)

func init() {
	Web = os.DirFS("dist")
	I18n = os.DirFS("i18n")
}
