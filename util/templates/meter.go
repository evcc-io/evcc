package templates

var meterReversedParam = Param{
	Name: ParamReversed,
	Description: TextLanguage{
		DE: "Richtung vertauscht",
		EN: "Direction reversed",
	},
	Help: TextLanguage{
		DE: "Kehrt das Vorzeichen von Leistung und Strömen um",
		EN: "Inverts the sign of power and currents",
	},
	Type:    TypeBool,
	Default: "false",
}

func (t *Template) prepare(class Class) {
	if class != Meter {
		return
	}

	if _, p := t.ParamByName(ParamReversed); p.Name == ParamReversed {
		return
	}

	t.Params = append(t.Params, meterReversedParam)
}
