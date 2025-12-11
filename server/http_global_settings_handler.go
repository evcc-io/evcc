package server

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/redact"
	"github.com/gorilla/mux"
	"go.yaml.in/yaml/v4"
)

func settingsGetStringHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := settings.String(key)

		// Check if private data should be hidden
		if r.URL.Query().Get("private") == "false" && res != "" {
			res = redact.String(res)
		}

		jsonWrite(w, res)
	}
}

func settingsDeleteHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(key, "")
		jsonWrite(w, true)
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

		jsonWrite(w, val)
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

		jsonWrite(w, val)
	}
}

func settingsSetJsonHandler(key string, pub publisher, newStruc func() any) http.HandlerFunc {
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
			// Skip merge for slices - they should be replaced entirely
			if reflect.ValueOf(struc).Elem().Kind() != reflect.Slice {
				if err := mergeMaskedAny(oldStruc, struc); err != nil {
					jsonError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}

		settings.SetJson(key, struc)
		setConfigDirty()

		pub(key, struc)

		jsonWrite(w, true)
	}
}

func settingsDeleteJsonHandler(key string, pub publisher, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(key, "")
		setConfigDirty()

		pub(key, struc)

		jsonWrite(w, true)
	}
}
