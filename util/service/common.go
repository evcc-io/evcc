package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
)

// applyCast applies optional type casting
func applyCast(value any, castType string) any {
	switch strings.ToLower(castType) {
	case "int":
		return cast.ToInt64(value)
	case "float":
		return cast.ToFloat64(value)
	case "bool":
		return cast.ToBool(value)
	case "string":
		return cast.ToString(value)
	default:
		return value
	}
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
