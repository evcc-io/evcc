package server

import (
	"bytes"
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

func settingsSetYamlHandler(key string, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		other := make(map[string]any)
		if err := yaml.NewDecoder(bytes.NewBuffer(b)).Decode(&other); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := util.DecodeOther(other, &struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// var res strings.Builder
		// enc := yaml.NewEncoder(&res)
		// enc.SetIndent(2)

		// if err := enc.Encode(struc); err != nil {
		// 	jsonError(w, http.StatusBadRequest, err)
		// 	return
		// }

		// val := res.String()
		val := strings.TrimSpace(string(b))
		settings.SetString(key, val)
		setConfigDirty()

		w.WriteHeader(http.StatusOK)
		jsonResult(w, val)
	}
}

func settingsSetJsonHandler(key string, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		settings.SetJson(key, struc)
		setConfigDirty()

		w.WriteHeader(http.StatusOK)
		jsonResult(w, struc)
	}
}
