package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logstash"
)

// logLevelsMux keeps stored and active levels in sync across concurrent updates
var logLevelsMux sync.Mutex

// updateLogLevelsHandler persists the log levels and applies them without restart
func updateLogLevelsHandler(pub publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var conf globalconfig.LogLevels
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&conf); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if !logstash.ValidLevel(conf.Default) {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid log level: %s", conf.Default))
			return
		}
		conf.Default = strings.ToUpper(conf.Default)

		levels := make(map[string]string, len(conf.Levels))
		for area, level := range conf.Levels {
			if strings.TrimSpace(area) == "" || !logstash.ValidLevel(level) {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid log level for area %q: %s", area, level))
				return
			}
			levels[area] = strings.ToUpper(level)
		}
		conf.Levels = levels

		logLevelsMux.Lock()
		defer logLevelsMux.Unlock()

		if err := settings.SetJson(keys.LogLevels, conf); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		if err := settings.Persist(); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		util.LogLevel(conf.Default, conf.Levels)
		pub(keys.LogLevels, conf)

		jsonWrite(w, conf)
	}
}
