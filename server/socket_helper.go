package server

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
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

func encodeSlice(v interface{}) (string, error) {
	rv := reflect.ValueOf(v)
	res := make([]string, rv.Len())

	for i := 0; i < rv.Len(); i++ {
		var err error
		if res[i], err = encode(rv.Index(i).Interface()); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(res, ",")), nil
}

func kv(p util.Param) string {
	var (
		val string
		err error
	)

	// unwrap slices
	if p.Val != nil && reflect.TypeOf(p.Val).Kind() == reflect.Slice {
		val, err = encodeSlice(p.Val)
	} else {
		val, err = encode(p.Val)
	}

	if err != nil {
		panic(err)
	}

	if p.Key == "" && val == "" {
		log.ERROR.Printf("invalid key/val for %+v, please report to https://github.com/evcc-io/evcc/issues/6439", p)
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
