package i18n

import (
	"fmt"

	assets "github.com/evcc-io/evcc/assets/i18n"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Config = i18n.LocalizeConfig

var localizer *i18n.Localizer

func Init() error {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	dir, err := assets.LocaleFS.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, d := range dir {
		if _, err := bundle.LoadMessageFileFS(assets.LocaleFS, d.Name()); err != nil {
			return fmt.Errorf("loading locales failed: %w", err)
		}
	}

	lang, err := jibber_jabber.DetectLanguage()
	if err != nil {
		lang = "de"
	}

	localizer = i18n.NewLocalizer(bundle, lang)

	return nil
}

func Localize(lc *Config) string {
	msg, _, err := localizer.LocalizeWithTag(lc)
	if err != nil {
		msg = lc.MessageID
	}
	return msg
}
