package templates

import (
	"reflect"
	"strings"
)

// https://github.com/peterbourgon/mergemap

const mergeMaxDepth = 100

var matchKey = strings.EqualFold

// mergeMaps recursively merges other into target using matchKey for key comparison
func mergeMaps(other map[string]any, target map[string]any) error {
	merge(target, other, 0)
	return nil
}

func merge(dst, src map[string]any, depth int) map[string]any {
	if depth > mergeMaxDepth {
		panic("too deep!")
	}
outer:
	for key, srcVal := range src {
		for k := range dst {
			if matchKey(k, key) {
				// empty value keeps the default
				if srcVal == "" {
					continue outer
				}

				// overwrite key
				key = k

				srcMap, srcMapOk := mapify(srcVal)
				dstMap, dstMapOk := mapify(dst[k])
				if srcMapOk && dstMapOk {
					srcVal = merge(dstMap, srcMap, depth+1)
				}
				break
			}
		}
		dst[key] = srcVal
	}
	return dst
}

func mapify(i any) (map[string]any, bool) {
	value := reflect.ValueOf(i)
	if value.Kind() == reflect.Map {
		m := map[string]any{}
		for _, k := range value.MapKeys() {
			m[k.String()] = value.MapIndex(k).Interface()
		}
		return m, true
	}
	return map[string]any{}, false
}
