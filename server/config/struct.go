package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structs"
)

type Descriptor struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Required bool          `json:"required"`
	Label    string        `json:"label"`
	Enum     []interface{} `json:"enum,omitempty"`
	Default  interface{}   `json:"default,omitempty"`
	Children []Descriptor  `json:"children,omitempty"`
}

// tagKey returns tag key's value or key name if value is empty
func tagKey(f *structs.Field, tag, key string) string {
	keyvals := strings.Split(f.Tag(tag), ",")

	for _, kv := range keyvals {
		if splits := strings.SplitN(kv, "=", 2); splits[0] == key {
			if len(splits) > 1 {
				return splits[1]
			}
			return key
		}
	}

	return ""
}

// enum converts list of strings to enum values
func enum(list []string) (enum []interface{}) {
	for _, v := range list {
		enum = append(enum, strings.TrimSpace(v))
	}
	return enum
}

// label is the exported field label
func label(f *structs.Field) string {
	val := f.Tag("ui")
	if val == "" {
		val = translate(f.Name())
	}
	if val == "" {
		val = f.Name()
	}

	return val
}

// kind is the exported data type
func kind(f *structs.Field) string {
	switch f.Value().(type) {
	case time.Duration:
		return "duration"
	default:
		return f.Kind().String()
	}
}

// value kind is the exported default value
func value(f *structs.Field) interface{} {
	switch val := f.Value().(type) {
	case time.Duration:
		return val / time.Second
	default:
		return f.Value()
	}
}

func Describe(s interface{}, opt ...bool) (ds []Descriptor) {
	var flat bool
	if len(opt) == 1 && opt[0] {
		flat = true
	}

	for _, f := range structs.Fields(s) {
		if !f.IsExported() {
			continue
		}

		// embedded fields
		if f.Kind() == reflect.Struct && f.IsEmbedded() {
			dd := Describe(f.Value())
			ds = append(ds, dd...)
			continue
		}

		// normal fields including structs
		d := Descriptor{
			Name:     f.Name(),
			Type:     kind(f),
			Required: tagKey(f, "validate", "required") != "",
			Label:    label(f),
		}

		// enums
		if oneof := tagKey(f, "validate", "oneof"); oneof != "" {
			d.Enum = enum(strings.Split(oneof, " "))
		}

		if !f.IsZero() {
			d.Default = value(f)
		}

		if f.Kind() == reflect.Ptr {
			continue
		}
		if f.Kind() == reflect.Interface {
			continue
		}
		if f.Kind() == reflect.Func {
			continue
		}

		if f.Kind() == reflect.Struct {
			if flat {
				// don't describe the field
				continue
			}

			d.Default = nil // no default for structs
			d.Children = Describe(f.Value())
		}

		ds = append(ds, d)
	}

	return ds
}
