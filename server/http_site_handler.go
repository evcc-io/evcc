package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/gorilla/mux"
	"github.com/itchyny/gojq"
	"golang.org/x/text/language"
)

var ignoreState = []string{"releaseNotes"} // excessive size

// getPreferredLanguage returns the preferred language as two letter code
func getPreferredLanguage(header string) string {
	languages, _, err := language.ParseAcceptLanguage(header)
	if err != nil || len(languages) == 0 {
		return "en"
	}

	base, _ := languages[0].Base()
	return base.String()
}

func indexHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := fs.ReadFile(assets.Web, "index.html")
		if err != nil {
			log.FATAL.Print("httpd: failed to load embedded template:", err.Error())
			log.FATAL.Print("Make sure templates are included using the `release` build tag or use `make build`")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		t, err := template.New("evcc").Delims("[[", "]]").Parse(string(indexTemplate))
		if err != nil {
			log.FATAL.Fatal("httpd: failed to create main page template:", err.Error())
		}

		defaultLang := getPreferredLanguage(r.Header.Get("Accept-Language"))

		if err := t.Execute(w, map[string]interface{}{
			"Version":     Version,
			"Commit":      Commit,
			"DefaultLang": defaultLang,
		}); err != nil {
			log.ERROR.Println("httpd: failed to render main page:", err.Error())
		}
	})
}

// jsonHandler is a middleware that decorates responses with JSON and CORS headers
func jsonHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		h.ServeHTTP(w, r)
	})
}

func jsonWrite(w http.ResponseWriter, content interface{}) {
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.ERROR.Printf("httpd: failed to encode JSON: %v", err)
	}
}

func jsonResult(w http.ResponseWriter, res interface{}) {
	jsonWrite(w, map[string]interface{}{"result": res})
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, map[string]interface{}{"error": err.Error()})
}

// pass converts a simple api without return value to api with nil error return value
func pass[T any](f func(T)) func(T) error {
	return func(v T) error {
		f(v)
		return nil
	}
}

// floatHandler updates float-param api
func floatHandler(set func(float64) error, get func() float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.ParseFloat(vars["value"], 64)
		if err == nil {
			err = set(val)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, get())
	}
}

// intHandler updates int-param api
func intHandler(set func(int) error, get func() int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.Atoi(vars["value"])
		if err == nil {
			err = set(val)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, get())
	}
}

// boolHandler updates bool-param api
func boolHandler(set func(bool) error, get func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.ParseBool(vars["value"])
		if err == nil {
			err = set(val)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, get())
	}
}

// boolGetHandler retrieves bool api values
func boolGetHandler(get func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonResult(w, get())
	}
}

// encodeFloats replaces NaN and Inf with nil
// TODO handle hierarchical data
func encodeFloats(data map[string]any) {
	for k, v := range data {
		switch v := v.(type) {
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				data[k] = nil
			}
		}
	}
}

// stateHandler returns the combined state
func stateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		for _, k := range ignoreState {
			delete(res, k)
		}

		encodeFloats(res)

		if q := r.URL.Query().Get("jq"); q != "" {
			q = strings.TrimPrefix(q, ".result")

			query, err := gojq.Parse(q)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			b, err := json.Marshal(res)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			res, err := jq.Query(query, b)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			jsonWrite(w, res)
			return
		}

		jsonResult(w, res)
	}
}

// healthHandler returns current charge mode
func healthHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if site == nil || !site.Healthy() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}

// tariffHandler returns the configured tariff
func tariffHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tariff := vars["tariff"]

		t := site.GetTariff(tariff)
		if t == nil {
			jsonError(w, http.StatusNotFound, errors.New("tariff not available"))
			return
		}

		rates, err := t.Rates()
		if err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}

		res := struct {
			Rates api.Rates `json:"rates"`
		}{
			Rates: rates,
		}

		jsonResult(w, res)
	}
}

// socketHandler attaches websocket handler to uri
func socketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWebsocket(w, r)
	}
}
