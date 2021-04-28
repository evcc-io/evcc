package mercedes

import (
	"encoding/json"
	"strconv"
)

type IntVal int

func (si *IntVal) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return err
	}

	*si = IntVal(n)
	return nil
}
