package logstash

import (
	"log/slog"
	"regexp"
	"slices"
)

type element string

var re = regexp.MustCompile(`^\[(.+?)\s*\] (\w+) `)

func (e element) areaLevel() (string, slog.Level) {
	m := re.FindAllStringSubmatch(string(e), 1)
	if len(m) != 1 || len(m[0]) != 3 {
		return "", slog.LevelError
	}
	return m[0][1], LogLevelToThreshold(m[0][2])
}

func (e element) match(areas []string, level slog.Level) bool {
	a, l := e.areaLevel()
	return (len(areas) == 0 || slices.Contains(areas, a)) && l >= level
}
