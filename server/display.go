package server

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

//go:embed display/static/*
var displayFiles embed.FS

// NewDisplayHTTPd creates the read-only garage display server.
func NewDisplayHTTPd(addr string, cache *util.ParamCache) *HTTPd {
	display, err := fs.Sub(displayFiles, "display/static")
	if err != nil {
		panic(fmt.Errorf("display assets: %w", err))
	}

	router := mux.NewRouter()
	router.Handle("/api/state", http.HandlerFunc(stateHandler(cache))).Methods(http.MethodGet)
	router.HandleFunc("/fonts/{name:Montserrat-(?:Medium|Bold)\\.woff2}", displayFontHandler).Methods(http.MethodGet)

	files := http.FileServer(http.FS(display))
	router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))

	log := util.NewLogger("display")
	srv := &HTTPd{
		Server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			ErrorLog:     log.ERROR,
		},
	}
	srv.SetKeepAlivesEnabled(true)

	return srv
}

func displayFontHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if assets.Web == nil {
		http.NotFound(w, r)
		return
	}

	fontName := strings.TrimSuffix(name, ".woff2")
	matches, err := fs.Glob(assets.Web, "assets/"+fontName+"-*.woff2")
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	if len(matches) != 1 {
		http.NotFound(w, r)
		return
	}

	font, err := assets.Web.Open(matches[0])
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	defer font.Close()

	stat, err := fs.Stat(assets.Web, matches[0])
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	seeker, ok := font.(io.ReadSeeker)
	if !ok {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, name, stat.ModTime(), seeker)
}
