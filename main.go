package main

import (
	"embed"
	"io"
	"io/fs"
	"log"

	"github.com/evcc-io/evcc/cmd"
	"github.com/evcc-io/evcc/server/assets"
	_ "golang.org/x/crypto/x509roots/fallback" // fallback certificates
)

var (
	//go:embed dist
	web embed.FS

	//go:embed i18n/*.json
	i18n embed.FS
)

// init loads embedded assets unless live assets are already loaded
func init() {
	if !assets.Live() {
		var err error

		assets.Web, err = fs.Sub(web, "dist")
		if err != nil {
			panic(err)
		}

		assets.I18n, err = fs.Sub(i18n, "i18n")
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	// suppress deprecated: golang.org/x/oauth2: Transport.CancelRequest no longer does anything; use contexts
	// see https://github.com/golang/oauth2/issues/487
	log.SetOutput(io.Discard)

	cmd.Execute()
}
