package main

import (
	"embed"
	"io/fs"

	"github.com/andig/evcc/cmd"
	"github.com/andig/evcc/server"
)

//go:embed dist
var assets embed.FS

// init loads embedded assets unless live assets are already loaded
func init() {
	if server.Assets == nil {
		fsys, err := fs.Sub(assets, "dist")
		if err != nil {
			panic(err)
		}
		server.Assets = fsys
	}
}

func main() {
	cmd.Execute()
}
