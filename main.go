package main

import (
	"embed"
	"io"
	"io/fs"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/jibber_jabber"
	lang "github.com/evcc-io/evcc/assets/i18n"
	"github.com/evcc-io/evcc/cmd"
	"github.com/evcc-io/evcc/server"
	_ "github.com/evcc-io/evcc/util/goversion" // require minimum go version
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
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
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	dir, err := lang.LocaleFS.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, d := range dir {
		if _, err := bundle.LoadMessageFileFS(lang.LocaleFS, d.Name()); err != nil {
			panic("loading " + d.Name() + " failed: " + err.Error())
		}
	}

	lang, err := jibber_jabber.DetectLanguage()
	if err != nil {
		lang = "de"
	}

	localizer := i18n.NewLocalizer(bundle, lang)
	s := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "footer.version.availableLong",
	})

	panic(s)

	// suppress deprecated: golang.org/x/oauth2: Transport.CancelRequest no longer does anything; use contexts
	// see https://github.com/golang/oauth2/issues/487
	log.SetOutput(io.Discard)

	cmd.Execute()
}
