package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/evcc-io/evcc/util/encode"
)

var enc = encode.NewEncoder(encode.WithDuration())

func encodeAsString(v any) (string, error) {
	b, err := json.Marshal(enc.Encode(v))
	return string(b), err
}

func encodeSliceAsString(v any) (string, error) {
	rv := reflect.ValueOf(v)
	res := make([]string, rv.Len())

	for i := range rv.Len() {
		var err error
		if res[i], err = encodeAsString(rv.Index(i).Interface()); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(res, ",")), nil
}

func socketEncode(pval any) string {
	var (
		val string
		err error
	)

	// unwrap slices
	if rv := reflect.ValueOf(pval); pval != nil && rv.Kind() == reflect.Slice && !rv.IsNil() {
		val, err = encodeSliceAsString(pval)
	} else {
		val, err = encodeAsString(pval)
	}

	if err != nil {
		panic(err)
	}

	return val
}
