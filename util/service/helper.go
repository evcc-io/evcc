package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
)

// toString converts to canonical string representation
func toString(value any, castType string) string {
	res := value
	switch strings.ToLower(castType) {
	case "int":
		res = cast.ToInt64(value)
	case "bool":
		res = cast.ToBool(value)
	case "float":
		return strconv.FormatFloat(cast.ToFloat64(value), 'g', 3, 64)
	}
	return cast.ToString(res)
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
