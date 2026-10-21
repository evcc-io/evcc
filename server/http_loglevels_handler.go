package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logstash"
)

// updateLogLevelsHandler persists the log levels and applies them without restart
func updateLogLevelsHandler(pub publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Default string             `json:"default"`
			Levels  *map[string]string `json:"levels"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		conf := globalconfig.LogLevels{Default: strings.ToUpper(req.Default)}
		if req.Levels != nil {
			conf.Levels = make(map[string]string, len(*req.Levels))
			for area, level := range *req.Levels {
				conf.Levels[area] = strings.ToUpper(level)
			}
		} else {
			// omitted overrides stay as they are
			_, conf.Levels = util.ActiveLogLevel()
		}

		if !logstash.ValidLevel(conf.Default) {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid log level: %s", conf.Default))
			return
		}

		for area, level := range conf.Levels {
			if strings.TrimSpace(area) == "" || !logstash.ValidLevel(level) {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid log level for area %q: %s", area, level))
				return
			}
		}

		if err := settings.SetJson(keys.LogLevels, conf); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		setConfigDirty()

		util.LogLevel(conf.Default, conf.Levels)
		publishLogLevels(pub)

		jsonWrite(w, true)
	}
}

// deleteLogLevelsHandler removes the persisted log levels and falls back to the command line defaults
func deleteLogLevelsHandler(pub publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := settings.Delete(keys.LogLevels); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		setConfigDirty()

		util.ResetLogLevel()
		publishLogLevels(pub)

		jsonWrite(w, true)
	}
}

func publishLogLevels(pub publisher) {
	level, levels := util.ActiveLogLevel()
	pub(keys.LogLevels, globalconfig.LogLevels{Default: level, Levels: levels})
}
