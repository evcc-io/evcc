package config

import "github.com/evcc-io/evcc/util/templates"

// DeviceRefs returns the devices referenced by a template-based device config, keyed by class
func DeviceRefs(class templates.Class, conf Named) map[templates.Class][]string {
	name, _ := conf.Property("template").(string)
	if conf.Type != "template" || name == "" {
		return nil
	}

	tmpl, err := templates.ByName(class, name)
	if err != nil {
		return nil
	}

	return tmpl.DeviceRefs(conf.Other)
}
