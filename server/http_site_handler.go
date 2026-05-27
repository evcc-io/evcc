package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/encode"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/logstash"
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

func indexHandler(customCss bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")

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

		if err := t.Execute(w, map[string]any{
			"Version":     util.Version,
			"Commit":      util.Commit,
			"DefaultLang": defaultLang,
			"CustomCss":   customCss,
		}); err != nil {
			log.ERROR.Println("httpd: failed to render main page:", err.Error())
		}
	}
}

// jsonHandler is a middleware that decorates responses with JSON and CORS headers
func jsonHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		h.ServeHTTP(w, r)
	})
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}

func handler[T any](conv func(string) (T, error), set func(T) error, get func() T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := conv(vars["value"])
		if err == nil {
			err = set(val)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, get())
	}
}

// ptrHandler updates pointer api
func ptrHandler[T any](conv func(string) (T, error), set func(*T) error, get func() *T) http.HandlerFunc {
	return handler(func(s string) (*T, error) {
		var val *T
		v, err := conv(s)
		if err == nil {
			val = &v
		} else if s == "" {
			err = nil
		}
		return val, err
	}, set, get)
}

// floatHandler updates float-param api
func floatHandler(set func(float64) error, get func() float64) http.HandlerFunc {
	return handler(parseFloat, set, get)
}

// floatPtrHandler updates float-pointer api
func floatPtrHandler(set func(*float64) error, get func() *float64) http.HandlerFunc {
	return ptrHandler(parseFloat, set, get)
}

// intHandler updates int-param api
func intHandler(set func(int) error, get func() int) http.HandlerFunc {
	return handler(strconv.Atoi, set, get)
}

// boolHandler updates bool-param api
func boolHandler(set func(bool) error, get func() bool) http.HandlerFunc {
	return handler(strconv.ParseBool, set, get)
}

// durationHandler updates duration-param api
func durationHandler(set func(time.Duration) error, get func() time.Duration) http.HandlerFunc {
	return handler(util.ParseDuration, set, get)
}

// getHandler returns api results
func getHandler[T any](get func() T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonWrite(w, get())
	}
}

// updateSmartCostLimit sets the smart cost limit globally
func updateSmartCostLimit(site site.API, setLimit func(loadpoint.API, *float64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var val *float64

		if r.Method != http.MethodDelete {
			f, err := parseFloat(vars["value"])
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			val = &f
		}

		for _, lp := range site.Loadpoints() {
			setLimit(lp, val)
		}

		jsonWrite(w, val)
	}
}

// updateBatteryMode sets the external battery mode
func updateBatteryMode(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var val api.BatteryMode

		if r.Method != http.MethodDelete {
			s, err := api.BatteryModeString(vars["value"])
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			val = s
		}

		site.SetBatteryModeExternal(val)

		jsonWrite(w, site.GetBatteryModeExternal())
	}
}

// stateHandler returns the combined state
func stateHandler(cache *util.ParamCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State(encode.NewEncoder(encode.WithDuration()))
		for _, k := range ignoreState {
			delete(res, k)
		}

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

		jsonWrite(w, res)
	}
}

// tariffHandler returns the configured tariff
func tariffHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		val := vars["tariff"]

		tariff, err := api.TariffUsageString(val)
		if err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}

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

		jsonWrite(w, res)
	}
}

// socketHandler attaches websocket handler to uri
func socketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWebsocket(w, r)
	}
}

func logAreasHandler(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, logstash.Areas())
}

func clearCacheHandler(w http.ResponseWriter, r *http.Request) {
	util.ResetCached()
	jsonWrite(w, "OK")
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	a := r.URL.Query()["area"]
	l := logstash.LogLevelToThreshold(r.URL.Query().Get("level"))

	var count int
	if v := r.URL.Query().Get("count"); v != "" {
		count, _ = strconv.Atoi(v)
	}

	log := logstash.All(a, l, count)

	if r.URL.Query().Get("format") == "txt" {
		filename := "evcc-" + time.Now().Format("20060102-150405") + `-` + strings.ToLower(l.String()) + ".log"
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

		for _, s := range log {
			if _, err := w.Write([]byte(s)); err != nil {
				return
			}
		}

		return
	}

	jsonWrite(w, log)
}

// adminPasswordValid validates the admin password and returns true if valid
func adminPasswordValid(authObject auth.Auth, password string) bool {
	return authObject.GetAuthMode() == auth.Disabled || authObject.IsAdminPasswordValid(password)
}

func getBackup(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !adminPasswordValid(authObject, req.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		if err := settings.Persist(); err != nil {
			http.Error(w, "Synching DB failed", http.StatusInternalServerError)
			return
		}

		filename := "evcc-backup-" + time.Now().Format("2006-01-02--15-04") + ".db"

		tmpFile, err := os.CreateTemp("", "evcc-backup-*.db")
		if err != nil {
			http.Error(w, "Creating backup failed", http.StatusInternalServerError)
			return
		}
		tmpName := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpName)

		if err := db.Backup(r.Context(), tmpName); err != nil {
			http.Error(w, "Backup failed", http.StatusInternalServerError)
			return
		}

		f, err := os.Open(tmpName)
		if err != nil {
			http.Error(w, "Opening backup failed", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, "Streaming backup failed", http.StatusInternalServerError)
			return
		}
	}
}

// createLocalDatabaseBackup creates a local backup in case of catastrophic error in reset or restore
func createLocalDatabaseBackup(ctx context.Context) error {
	return db.Backup(ctx, db.FilePath()+".bak")
}

func restoreDatabase(authObject auth.Auth, shutdown func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// cap upload size to bound disk usage
		r.Body = http.MaxBytesReader(w, r.Body, 256<<20)

		mr, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		var (
			password string
			tmpName  string
		)

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, "Upload failed", http.StatusBadRequest)
				return
			}

			switch part.FormName() {
			case "password":
				b, err := io.ReadAll(io.LimitReader(part, 1<<10))
				part.Close()
				if err != nil {
					http.Error(w, "Upload failed", http.StatusBadRequest)
					return
				}
				password = string(b)

			case "file":
				tmpFile, err := os.CreateTemp("", "evcc-restore-*.db")
				if err != nil {
					part.Close()
					http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
					return
				}
				tmpName = tmpFile.Name()
				defer os.Remove(tmpName)

				_, copyErr := io.Copy(tmpFile, part)
				closeErr := tmpFile.Close()
				part.Close()
				if copyErr != nil || closeErr != nil {
					http.Error(w, "Upload failed", http.StatusBadRequest)
					return
				}

			default:
				part.Close()
			}
		}

		if !adminPasswordValid(authObject, password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		if tmpName == "" {
			http.Error(w, "Missing file", http.StatusBadRequest)
			return
		}

		settings.Persist()

		// create local backup before overwriting
		if err := createLocalDatabaseBackup(r.Context()); err != nil {
			http.Error(w, "Backup failed", http.StatusInternalServerError)
			return
		}

		if err := db.Restore(r.Context(), tmpName); err != nil {
			http.Error(w, "Restore failed", http.StatusInternalServerError)
			return
		}

		// close DB so the WAL is checkpointed into the main file before shutdown
		// hooks (e.g. settings.Persist) can overwrite the restored content
		if err := db.Close(); err != nil {
			http.Error(w, "DB close failed", http.StatusInternalServerError)
			return
		}

		shutdown()
		w.WriteHeader(http.StatusNoContent)
	}
}

func resetDatabase(authObject auth.Auth, shutdown func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
			Sessions bool   `json:"sessions"`
			Settings bool   `json:"settings"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if !adminPasswordValid(authObject, req.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		settings.Persist()

		if err := createLocalDatabaseBackup(r.Context()); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		if req.Sessions {
			query := db.Instance.Exec("DELETE FROM sessions")
			if query.Error != nil {
				jsonError(w, http.StatusInternalServerError, query.Error)
				return
			}
		}

		if req.Settings {
			tables := []string{"settings", "configs", "caches", "meters"}

			for _, table := range tables {
				if err := db.Instance.Exec("DELETE FROM " + table).Error; err != nil {
					jsonError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}

		// close db connection to avoid on-shutdown writes
		if err := db.Close(); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		shutdown()
		w.WriteHeader(http.StatusNoContent)
	}
}
