package logstash

import (
	"regexp"
	"slices"
)

type element string

var re = regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\s*\] (\w+) `)

func (e element) areaLevel() (string, string) {
	m := re.FindAllStringSubmatch(string(e), 1)
	if len(m) != 1 && len(m[0]) != 3 {
		return "", ""
	}
	return m[0][1], m[0][2]
}

func (e element) match(areas, levels []string) bool {
	a, l := e.areaLevel()
	return (len(areas) == 0 || slices.Contains(areas, a)) && (len(levels) == 0 || slices.Contains(levels, l))
}
