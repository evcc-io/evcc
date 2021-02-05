// +build !release

package server

import (
	"os"
)

func init() {
	Assets = os.DirFS("dist")
}
