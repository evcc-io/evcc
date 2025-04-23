package locale

import (
	"encoding/json"
	"fmt"
	"io/fs"

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
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	dir, err := fs.ReadDir(assets.I18n, ".")
	if err != nil {
		panic(err)
	}

	for _, d := range dir {
		name := d.Name()

		data, err := fs.ReadFile(assets.I18n, name)
		if err != nil {
			return fmt.Errorf("loading locale file %s failed: %w", name, err)
		}

		if _, err := Bundle.ParseMessageFileBytes(data, name); err != nil {
			return fmt.Errorf("parsing locale file %s failed: %w", name, err)
		}
	}

	// Detect system language, fallback to German if detection fails
	lang, err := jibber_jabber.DetectLanguage()
	if err != nil {
		Language = language.German.String()
	} else {
		Language = lang
	}

	// Initialize the localizer with the detected language
	Localizer = i18n.NewLocalizer(Bundle, Language)
	return nil
}
