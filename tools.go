//go:build tools

package main

import (
	_ "github.com/dmarkham/enumer"
	_ "github.com/evcc-io/evcc/cmd/tools"
	_ "go.uber.org/mock/mockgen"
)
