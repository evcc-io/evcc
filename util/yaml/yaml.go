package yaml

import (
	"strings"

	goyaml "go.yaml.in/yaml/v4"
)

func Marshal(data any) ([]byte, error) {
	return goyaml.Marshal(data)
}

// Unmarshal invokes unmarshaler, ignoring empty document errors
func Unmarshal(b []byte, res any) error {
	err := goyaml.Unmarshal(b, res)
	if err != nil && strings.Contains(err.Error(), "no documents in stream") {
		err = nil
	}
	return err
}
