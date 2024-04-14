package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/evcc-io/evcc/util"
)

func encode(v any) (string, error) {
	if b, err := json.Marshal(util.EncodeAny(v)); err == nil {
		return string(b), nil
	} else {
		return "", err
	}
}

func encodeSlice(v any) (string, error) {
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
		if p.Key == "interval" {
			p.Key = "interval"
		}
		if p.Key == "chargeDuration" {
			p.Key = "chargeDuration"
		}
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
