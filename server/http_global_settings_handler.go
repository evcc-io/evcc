package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/fatih/structs"
	"github.com/go-viper/mapstructure/v2"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func settingsGetStringHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := settings.String(key)
		jsonResult(w, res)
	}
}

func settingsDeleteHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(key, "")
		jsonResult(w, true)
	}
}

func settingsSetDurationHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		settings.SetInt(key, int64(time.Second*time.Duration(val)))
		setConfigDirty()

		jsonResult(w, val)
	}
}

func settingsSetYamlHandler(key string, other, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := yaml.NewDecoder(bytes.NewBuffer(b)).Decode(&other); err != nil && err != io.EOF {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := settings.DecodeOtherSliceOrMap(other, &struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		val := strings.TrimSpace(string(b))
		settings.SetString(key, val)
		setConfigDirty()

		jsonResult(w, val)
	}
}

func settingsSetJsonHandler(key string, valueChan chan<- util.Param, newStruc func() any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		struc := newStruc()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		oldStruc := newStruc()
		if err := settings.Json(key, &oldStruc); err == nil {
			if err := mergeSettings(oldStruc, struc); err != nil {
				jsonError(w, http.StatusInternalServerError, err)
				return
			}
		}

		settings.SetJson(key, struc)
		setConfigDirty()

		valueChan <- util.Param{Key: key, Val: struc}

		jsonResult(w, true)
	}
}

func settingsDeleteJsonHandler(key string, valueChan chan<- util.Param, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(key, "")
		setConfigDirty()

		valueChan <- util.Param{Key: key, Val: struc}

		jsonResult(w, true)
	}
}

func mergeSettings(old any, new any) error {
	redactable, ok := old.(api.Redactor)
	if !ok {
		return nil
	}

	newMap := structs.Map(new)
	oldMap := structs.Map(old)
	redactedMap := structs.Map(redactable.Redacted())

	for k, v := range newMap {
		if rv, ok := redactedMap[k]; ok && v == rv {
			if ov, ok := oldMap[k]; ok {
				newMap[k] = ov
			}
		}
	}

	return mapstructure.Decode(newMap, &new)
}
