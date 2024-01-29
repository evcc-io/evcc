package server

import (
	"io"
	"net/http"
	"os"
)

func yamlHandler(configFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Content  string `json:"content"`
			Writable bool   `json:"writable"`
			Path     string `json:"path"`
		}

		// read content from file, check if file is writable
		content, err := os.ReadFile(configFile)
		if err == nil {
			payload.Content = string(content)
		}
		// check if configFile is writable
		if info, err := os.Stat(configFile); err == nil {
			payload.Writable = info.Mode().Perm()&0200 != 0
		}
		payload.Path = configFile
		jsonResult(w, payload)
	}
}

func updateYamlHandler(configFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(configFile, body, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
