package util

func TypeWithTemplateName(typ string, other map[string]any) string {
	if template := TemplateName(typ, other); template != "" {
		typ += ":" + template
	}
	return typ
}

func TemplateName(typ string, other map[string]any) string {
	if typ == "template" && other != nil {
		if template, ok := other["template"].(string); ok && template != "" {
			return template
		}
	}
	return ""
}
