//go:build tools

package main

import (
	_ "github.com/dmarkham/enumer"
	_ "go.uber.org/mock/mockgen"
	_ "mvdan.cc/gofumpt"
)
