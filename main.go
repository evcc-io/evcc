package main

import (
	"embed"
	"io"
	"io/fs"
	"log"

	"github.com/evcc-io/evcc/cmd"
	"github.com/evcc-io/evcc/server"
	_ "github.com/evcc-io/evcc/util/goversion" // require minimum go version
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
	// suppress deprecated: golang.org/x/oauth2: Transport.CancelRequest no longer does anything; use contexts
	// see https://github.com/golang/oauth2/issues/487
	log.SetOutput(io.Discard)

	cmd.Execute()
}
