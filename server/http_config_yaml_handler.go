package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/evcc-io/evcc/util/redact"
)

// configYamlHandler returns the redacted evcc.yaml configuration file
func configYamlHandler(configFilePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use the provided config file path
		if configFilePath == "" {
			jsonError(w, http.StatusNotFound, fmt.Errorf("no config file found"))
			return
		}

		// Read the config file
		src, err := os.ReadFile(configFilePath)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to read config file: %w", err))
			return
		}

		// Redact sensitive information
		redacted := redact.String(string(src))

		// Return the redacted content as plain text
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(redacted))
	}
}
