package logstash

import (
	"slices"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
)

type element string

// areaLevel parses the `[area  ] LEVEL ` header. Lines without one count as error
// level without area. Runs on every Write, hence no regexp.
func (e element) areaLevel() (string, jww.Threshold) {
	s, ok := strings.CutPrefix(string(e), "[")
	if !ok {
		return "", jww.LevelError
	}

	area, rest, ok := strings.Cut(s, "] ")
	if !ok || area == "" {
		return "", jww.LevelError
	}

	level, _, ok := strings.Cut(rest, " ")
	if !ok {
		return "", jww.LevelError
	}

	return strings.TrimRight(area, " "), LogLevelToThreshold(level)
}

func (e element) match(areas []string, level jww.Threshold) bool {
	a, l := e.areaLevel()
	return (len(areas) == 0 || slices.Contains(areas, a)) && l >= level
}
