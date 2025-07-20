package logstash

import (
	"regexp"
	"slices"

	jww "github.com/spf13/jwalterweatherman"
)

type element string

var re = regexp.MustCompile(`^\[(.+?)\s*\] (\w+) `)

func (e element) areaLevel() (string, jww.Threshold) {
	m := re.FindAllStringSubmatch(string(e), 1)
	if len(m) != 1 || len(m[0]) != 3 {
		return "", jww.LevelError
	}
	return m[0][1], LogLevelToThreshold(m[0][2])
}

func (e element) match(areas []string, level jww.Threshold) bool {
	a, l := e.areaLevel()
	return (len(areas) == 0 || slices.Contains(areas, a)) && l >= level
}
