package server

import "encoding/json"

type product struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

type products []product

func (p products) MarshalJSON() (out []byte, err error) {
	if p == nil {
		return []byte(`null`), nil
	}
	if len(p) == 0 {
		return []byte(`{}`), nil
	}

	out = append(out, '{')
	for _, e := range p {
		key, err := json.Marshal(e.Name)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(e.Template)
		if err != nil {
			return nil, err
		}
		out = append(out, key...)
		out = append(out, ':')
		out = append(out, val...)
		out = append(out, ',')
	}

	// replace last ',' with '}'
	if len(out) > 1 {
		out[len(out)-1] = '}'
	} else {
		out = append(out, '}')
	}

	return out, nil
}
