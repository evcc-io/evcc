package easee

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts", getAccounts)

	service.Register("easee", mux)
}

func getAccounts(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(KnownAccounts())
}
