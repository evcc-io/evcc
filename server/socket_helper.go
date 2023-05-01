package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/kr/pretty"
)

func encode(v interface{}) (string, error) {
	var s string
	switch val := v.(type) {
	case time.Time:
		if val.IsZero() {
			s = "null"
		} else {
			s = fmt.Sprintf(`"%s"`, val.Format(time.RFC3339))
		}
	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		s = fmt.Sprintf("%d", int64(val.Seconds()))
	case float64:
		if math.IsNaN(val) {
			s = "null"
		} else {
			s = fmt.Sprintf("%.5g", val)
		}
	default:
		if b, err := json.Marshal(v); err == nil {
			s = string(b)
		} else {
			return "", err
		}
	}
	return s, nil
}

func kv(p util.Param) string {
	val, err := encode(p.Val)
	if err != nil {
		panic(err)
	}

	if p.Key == "" && val == "" {
		log.ERROR.Printf("invalid key/val for %+v %# v, please report to https://github.com/evcc-io/evcc/issues/6439", p, pretty.Formatter(p.Val))
		return "\"foo\":\"bar\""
	}

	var msg strings.Builder
	msg.WriteString("\"")
	if p.Loadpoint != nil {
		msg.WriteString(fmt.Sprintf("loadpoints.%d.", *p.Loadpoint))
	}
	msg.WriteString(p.Key)
	msg.WriteString("\":")
	msg.WriteString(val)

	return msg.String()
}
