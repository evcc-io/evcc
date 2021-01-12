package config

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/fatih/structs"
)

type Descriptor struct {
	Name     string       `json:"name"`
	Kind     string       `json:"kind"`
	Default  interface{}  `json:"default"`
	Required bool         `json:"required"`
	Children []Descriptor `json:"children"`
}

func (d Descriptor) MarshalJSON() ([]byte, error) {
	if len(d.Children) == 0 {
		m := struct {
			Name     string      `json:"name"`
			Kind     string      `json:"kind"`
			Required bool        `json:"required"`
			Default  interface{} `json:"default"`
		}{
			d.Name,
			d.Kind,
			d.Required,
			d.Default,
		}
		return json.Marshal(m)
	}

	shadow := struct {
		Name     string       `json:"name"`
		Kind     string       `json:"kind"`
		Required bool         `json:"required"`
		Default  interface{}  `json:"default"`
		Children []Descriptor `json:"children"`
	}{
		Name:     d.Name,
		Kind:     d.Kind,
		Required: d.Required,
		Default:  d.Default,
		Children: d.Children,
	}

	return json.Marshal(shadow)
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
func formatValue(f *structs.Field) interface{} {
	switch f.Kind() {
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
			Kind:     f.Kind().String(),
			Default:  formatValue(f),
			Required: hasTag(f, "validate", "required"),
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

			d.Default = "" // no default for structs
			d.Children = Describe(f.Value())
		}

		ds = append(ds, d)
	}

	return ds
}
