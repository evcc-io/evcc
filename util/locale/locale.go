package locale

import (
	"fmt"
	"io/fs"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util/locale/internal"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Config = i18n.LocalizeConfig

var (
	Locale internal.ContextKey

	Bundle    *i18n.Bundle
	Language  string
	Localizer *i18n.Localizer
)

func Init() error {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	dir, err := fs.ReadDir(assets.I18n, ".")
	if err != nil {
		panic(err)
	}

	for _, d := range dir {
		if _, err := Bundle.LoadMessageFileFS(assets.I18n, d.Name()); err != nil {
			return fmt.Errorf("loading locales failed: %w", err)
		}
	}

	Language, err = jibber_jabber.DetectLanguage()
	if err != nil {
		Language = language.German.String()
	}

	Localizer = i18n.NewLocalizer(Bundle, Language)

	return nil
}

func Localize(lc *Config) string {
	msg, _, err := Localizer.LocalizeWithTag(lc)
	if err != nil {
		msg = lc.MessageID
	}
	return msg
}

func LocalizeID(id string) string {
	return Localize(&Config{
		MessageID: id,
	})
}
