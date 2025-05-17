package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
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

		if err := yaml.Unmarshal(b, &other); err != nil {
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
			if err := mergeMaskedAny(oldStruc, struc); err != nil {
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
