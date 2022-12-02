//go:build !release

package server

import (
	"os"
)

func init() {
	Assets = os.DirFS("dist")
	I18n = os.DirFS("i18n")
}
