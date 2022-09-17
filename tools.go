//go:build tools

package main

import (
	_ "github.com/deepmap/oapi-codegen/cmd/oapi-codegen"
	_ "github.com/dmarkham/enumer"
	_ "github.com/golang/mock/mockgen"
)
