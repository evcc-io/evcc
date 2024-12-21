//go:build tools

package main

import (
	_ "github.com/dmarkham/enumer"
	_ "github.com/evcc-io/evcc/cmd/decorate"
	_ "go.uber.org/mock/mockgen"
)
