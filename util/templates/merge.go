package templates

import (
	"fmt"
	"reflect"
	"strings"
)

// https://github.com/peterbourgon/mergemap

const mergeMaxDepth = 100

var matchKey = strings.EqualFold

// mergeMaps recursively merges other into target using matchKey for key comparison
func mergeMaps(other map[string]any, target map[string]any) error {
	if key, ok := ambiguousKey(other); ok {
		return fmt.Errorf("ambiguous key: %s", key)
	}
	merge(target, other, 0)
	return nil
}

// ambiguousKey finds two keys that matchKey treats as equal. merge visits src in map
// order, so such keys would make the rendered result depend on iteration order
func ambiguousKey(m map[string]any) (string, bool) {
	for key, val := range m {
		for other := range m {
			if other != key && matchKey(other, key) {
				return key, true
			}
		}
		if sub, ok := mapify(val); ok {
			if key, ok := ambiguousKey(sub); ok {
				return key, true
			}
		}
	}
	return "", false
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
