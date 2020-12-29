package main

import (
	"embed"
	"io/fs"

	"github.com/andig/evcc/cmd"
	"github.com/andig/evcc/server"
)

//go:embed dist
var assets embed.FS

func init() {
	// use embedded assets unless live assets are already loaded
	if server.Assets == nil {
		fsys, _ := fs.Sub(assets, "dist")
		server.Assets = fsys
	}
}

func main() {
	cmd.Execute()
}
