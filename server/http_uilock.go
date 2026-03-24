package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/uilock"
)

// CurrentUILockConfig returns DB-backed UI lock settings when present, otherwise fallback (typically from YAML / startup merge).
func CurrentUILockConfig(fallback globalconfig.UILock) globalconfig.UILock {
	if settings.Exists(keys.UILock) {
		var u globalconfig.UILock
		if err := settings.Json(keys.UILock, &u); err == nil {
			if u.Timeout <= 0 {
				u.Timeout = globalconfig.DefaultUILockTimeout
			}
			return u
		}
	}
	if fallback.Timeout <= 0 {
		fallback.Timeout = globalconfig.DefaultUILockTimeout
	}
	return fallback
}

type uilockStatusResponse struct {
	Enabled         bool    `json:"enabled"`
	AppliesToClient bool    `json:"appliesToClient"`
	Unlocked        bool    `json:"unlocked"`
	PinConfigured   bool    `json:"pinConfigured"`
	Timeout         float64 `json:"timeout"`
}

func uilockStatusHandler(m *uilock.Manager, current func() globalconfig.UILock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := current()
		applies := m.Applies(r, cfg)
		unlocked := !applies
		if applies {
			unlocked = m.RefreshUnlockCookieIfValid(w, r, cfg)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(uilockStatusResponse{
			Enabled:         cfg.Enabled,
			AppliesToClient: applies,
			Unlocked:        unlocked,
			PinConfigured:   m.PinConfigured(),
			Timeout:         cfg.Timeout.Seconds(),
		})
	}
}

type uilockUnlockRequest struct {
	Pin string `json:"pin"`
}

func uilockUnlockHandler(m *uilock.Manager, current func() globalconfig.UILock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req uilockUnlockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !m.IsPinValid(req.Pin) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		cfg := current()
		if err := m.IssueUnlockCookie(w, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func uilockLockHandler(w http.ResponseWriter, r *http.Request) {
	uilock.ClearUnlockCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func settingsSetUILockHandler(pub publisher, current func() globalconfig.UILock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req globalconfig.UILock
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var old globalconfig.UILock
		_ = settings.Json(keys.UILock, &old)

		old.Enabled = req.Enabled
		old.Timeout = req.Timeout
		old.IPs = req.IPs
		old.TrustedProxies = req.TrustedProxies
		old.Pin = ""

		mgr := uilock.NewManager()
		switch req.Pin {
		case "", uilock.MaskedPin:
			// unchanged
		default:
			if err := mgr.SetPin(req.Pin); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := settings.SetJson(keys.UILock, old); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setConfigDirty()

		pub(keys.UILock, mgr.Published(CurrentUILockConfig(current())))
		jsonWrite(w, true)
	}
}

func settingsDeleteUILockHandler(pub publisher, current func() globalconfig.UILock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defaults := globalconfig.DefaultUILock()
		if err := settings.SetJson(keys.UILock, defaults); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		settings.SetString(keys.UiLockPin, "")
		setConfigDirty()

		mgr := uilock.NewManager()
		pub(keys.UILock, mgr.Published(CurrentUILockConfig(current())))
		jsonWrite(w, true)
	}
}
