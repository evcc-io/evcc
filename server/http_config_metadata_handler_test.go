package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestProductsHandlerTariffUsage(t *testing.T) {
	templatesOf := func(usage string) []string {
		r := httptest.NewRequest(http.MethodGet, "/products/tariff?usage="+usage, nil)
		r = mux.SetURLVars(r, map[string]string{"class": "tariff"})
		w := httptest.NewRecorder()
		productsHandler(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var res products
		require.NoError(t, json.NewDecoder(w.Body).Decode(&res))

		var names []string
		for _, p := range res {
			names = append(names, p.Template)
		}
		return names
	}

	grid := templatesOf("grid")
	require.True(t, slices.Contains(grid, "tibber"))
	require.True(t, slices.Contains(grid, "fixed"))
	require.False(t, slices.Contains(grid, "bkw"))

	feedin := templatesOf("feedin")
	require.True(t, slices.Contains(feedin, "bkw"))
	require.True(t, slices.Contains(feedin, "fixed"))
	require.False(t, slices.Contains(feedin, "tibber"))
}
