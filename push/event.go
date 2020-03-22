package push

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/andig/evcc/api"
)

var log = api.NewLogger("push")

type Event struct {
	Event      string
	Sender     string
	Attributes map[string]interface{}
}

func (e Event) Apply(template string) (string, error) {
	return replaceFormatted(template, e.Attributes)
}

var re = regexp.MustCompile(`\${(\w+)(:([a-zA-Z0-9%.]+))?}`)

// replaceFormatted replaces all occurrances of ${key} with val from the kv map.
// All keys of kv must exist inside the string to apply replacements to
func replaceFormatted(s string, kv map[string]interface{}) (string, error) {
	matches := re.FindAllStringSubmatch(s, -1)

	for len(matches) > 0 {
		for _, m := range matches {
			key := m[1]
			val, ok := kv[key]
			if !ok {
				return "", errors.New("could not find match for " + m[0])
			}

			// apply format
			format := m[3]
			if format != "" {
				val = fmt.Sprintf(format, val)
			}

			// update string
			literalMatch := m[0]
			s = strings.ReplaceAll(s, literalMatch, fmt.Sprintf("%v", val))
		}

		// update matches
		matches = re.FindAllStringSubmatch(s, -1)
	}

	return s, nil
}
