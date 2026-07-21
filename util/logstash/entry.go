package logstash

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"log/slog"
)

// Entry is a single structured log record
type Entry struct {
	Time    time.Time
	Area    string
	Level   slog.Level
	Message string
	Attrs   map[string]string
}

// LevelString returns the evcc log level name
func LevelString(l slog.Level) string {
	switch {
	case l < slog.LevelDebug:
		return "TRACE"
	case l < slog.LevelInfo:
		return "DEBUG"
	case l < slog.LevelWarn:
		return "INFO"
	case l < slog.LevelError:
		return "WARN"
	case l < LevelFatal:
		return "ERROR"
	default:
		return "FATAL"
	}
}

// QuoteAttr quotes an attribute value if it contains special characters
func QuoteAttr(v string) string {
	if strings.ContainsAny(v, " \t\n\"=") {
		return strconv.Quote(v)
	}
	return v
}

func (e Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Time    time.Time         `json:"time"`
		Area    string            `json:"area"`
		Level   string            `json:"level"`
		Message string            `json:"message"`
		Attrs   map[string]string `json:"attrs,omitempty"`
	}{e.Time, e.Area, strings.ToLower(LevelString(e.Level)), e.Message, e.Attrs})
}

// String renders the entry in the console log line format
func (e Entry) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "[%-6s] %s %s %s", e.Area, LevelString(e.Level), e.Time.Format("2006/01/02 15:04:05"), e.Message)
	for _, k := range slices.Sorted(maps.Keys(e.Attrs)) {
		fmt.Fprintf(b, " %s=%s", k, QuoteAttr(e.Attrs[k]))
	}
	b.WriteByte('\n')
	return b.String()
}

func (e Entry) match(areas []string, level slog.Level) bool {
	return (len(areas) == 0 || slices.Contains(areas, e.Area)) && e.Level >= level
}
