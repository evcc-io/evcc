// +build tools

package main

import (
	_ "github.com/andig/evcc/cmd/tools/decorate"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/mjibson/esc"
	_ "golang.org/x/tools/cmd/stringer"
)
