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
		var data map[string]map[string]map[string]any
		if _, err := toml.DecodeFS(assets.I18n, d.Name(), &data); err != nil {
			return fmt.Errorf("loading locales failed: %w", err)
		}

		// load sessions.csv only
		if sessions := data["sessions"]; sessions != nil && len(sessions["csv"]) != 0 {
			b, err := toml.Marshal(map[string]any{
				"sessions": map[string]any{
					"csv": sessions["csv"],
				},
			})
			if err != nil {
				return fmt.Errorf("marshal session.csv failed: %w", err)
			}

			if _, err := Bundle.ParseMessageFileBytes(b, d.Name()); err != nil {
				return fmt.Errorf("loading locales failed: %w", err)
			}
		}
	}

	Language, err = jibber_jabber.DetectLanguage()
	if err != nil {
		Language = language.German.String()
	}

	Localizer = i18n.NewLocalizer(Bundle, Language)

	return nil
}
