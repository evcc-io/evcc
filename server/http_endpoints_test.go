package server

import (
	"net/http"
	"sort"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEndpointPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "normalizes route params with regex",
			path: "/api/vehicles/{name:[a-zA-Z0-9_.:-]+}/minsoc/{value:[0-9]+}",
			want: "/api/vehicles/{name}/minsoc/{value}",
		},
		{
			name: "normalizes numeric loadpoint id",
			path: "/api/loadpoints/12/mode/{value:[a-z]+}",
			want: "/api/loadpoints/{id}/mode/{value}",
		},
		{
			name: "keeps already normalized path",
			path: "/api/state",
			want: "/api/state",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeEndpointPath(tc.path))
		})
	}
}

func TestIsProtectedAPIPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "config path is protected", path: "/api/config/templates/charger", want: true},
		{name: "system path is protected", path: "/api/system/log", want: true},
		{name: "auth path is public", path: "/api/auth/status", want: false},
		{name: "state path is public", path: "/api/state", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isProtectedAPIPath(tc.path))
		})
	}
}

func TestListAPIEndpoints(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/state", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodGet)
	api.HandleFunc("/auth/status", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodGet)
	api.HandleFunc("/config/templates/{class:[a-z]+}", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodGet)
	api.HandleFunc("/system/log", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodGet)
	api.HandleFunc("/loadpoints/1/mode/{value:[a-z]+}", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodPost)
	api.HandleFunc("/loadpoints/2/mode/{value:[a-z]+}", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodPost)
	api.HandleFunc("/vehicles/{name:[a-zA-Z0-9_.:-]+}/minsoc/{value:[0-9]+}", func(http.ResponseWriter, *http.Request) {}).Methods(http.MethodPost)

	manifest, err := listAPIEndpoints(router)
	require.NoError(t, err)

	assert.True(t, sort.StringsAreSorted(manifest.Public))
	assert.True(t, sort.StringsAreSorted(manifest.Protected))

	assert.Contains(t, manifest.Public, "/api/state")
	assert.Contains(t, manifest.Public, "/api/auth/status")
	assert.Contains(t, manifest.Public, "/api/loadpoints/{id}/mode/{value}")
	assert.Contains(t, manifest.Public, "/api/vehicles/{name}/minsoc/{value}")

	assert.Contains(t, manifest.Protected, "/api/config/templates/{class}")
	assert.Contains(t, manifest.Protected, "/api/system/log")

	assert.NotContains(t, manifest.Public, "/health")
	assert.NotContains(t, manifest.Protected, "/health")
	assert.Equal(t, 1, endpointOccurrences(manifest.Public, "/api/loadpoints/{id}/mode/{value}"))
}

func endpointOccurrences(endpoints []string, value string) int {
	count := 0
	for _, endpoint := range endpoints {
		if endpoint == value {
			count++
		}
	}
	return count
}

