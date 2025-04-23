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

		// Unmarshal into generic structure to inspect sessions field
		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			return fmt.Errorf("unmarshal locale file %s failed: %w", d.Name(), err)
		}

		// Session-specific logic: load only sessions.csv if present
		if sessionsRaw, ok := raw["sessions"]; ok {
			if sessionsMap, ok2 := sessionsRaw.(map[string]interface{}); ok2 {
				if csvVal, ok3 := sessionsMap["csv"]; ok3 {
					// Marshal only the sessions.csv sub-tree back to JSON
					sub := map[string]interface{}{
						"sessions": map[string]interface{}{"csv": csvVal},
					}
					b2, err := json.Marshal(sub)
					if err != nil {
						return fmt.Errorf("marshal sessions for %s failed: %w", d.Name(), err)
					}
					if _, err := Bundle.ParseMessageFileBytes(b2, d.Name()); err != nil {
						return fmt.Errorf("loading session locales failed: %w", err)
					}
					continue
				}
			}
		}

		// Otherwise parse full JSON file as message file
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
