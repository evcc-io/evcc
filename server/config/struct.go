package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structs"
)

type Descriptor struct {
	Name     string       `json:"name"`
	Kind     string       `json:"kind"`
	Required bool         `json:"required"`
	Label    string       `json:"label"`
	Default  *interface{} `json:"default,omitempty"`
	Children []Descriptor `json:"children,omitempty"`
}

func hasTag(f *structs.Field, tag, key string) bool {
	tags := strings.Split(f.Tag(tag), ",")

	for _, v := range tags {
		if v == key {
			return true
		}
	}

	return false
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
			Kind:     kind(f),
			Required: hasTag(f, "validate", "required"),
			Label:    label(f),
		}

		if !f.IsZero() {
			def := value(f)
			d.Default = &def
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
