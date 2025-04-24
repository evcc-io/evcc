package locale

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

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

type sessions struct {
	Sessions struct {
		CSV json.RawMessage `json:"csv"`
	} `json:"sessions"`
}

// Init initializes the localization bundle and loads all JSON message files.
func Init() error {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	dir, err := fs.ReadDir(assets.I18n, ".")
	if err != nil {
		panic(err)
	}

	// Iterate over each file and process only .json files
	for _, d := range dir {
		if filepath.Ext(d.Name()) != ".json" {
			continue
		}

		b, err := fs.ReadFile(assets.I18n, d.Name())
		if err != nil {
			return fmt.Errorf("reading locale file %s failed: %w", d.Name(), err)
		}

		var s sessions
		err = json.Unmarshal(b, &s)
		if err == nil && len(s.Sessions.CSV) > 0 {
			sub := map[string]json.RawMessage{ "sessions": s.Sessions.CSV }
			b2, err := json.Marshal(sub)
			if err != nil {
				return fmt.Errorf("marshal sessions for %s failed: %w", d.Name(), err)
			}
			if _, err := Bundle.ParseMessageFileBytes(b2, d.Name()); err != nil {
				return fmt.Errorf("loading session locales failed: %w", err)
			}
			continue
		}

		if _, err := Bundle.ParseMessageFileBytes(b, d.Name()); err != nil {
			return fmt.Errorf("loading locales failed: %w", err)
		}
	}

	// Detect system language; default to German on failure
	Language, err = jibber_jabber.DetectLanguage()
	if err != nil {
		Language = language.German.String()
	}

	// Create a localizer for the detected language
	Localizer = i18n.NewLocalizer(Bundle, Language)

	return nil
}
