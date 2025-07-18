package server

import (
	_ "embed"
)

//go:embed openapi.yaml
var OpenAPI []byte
