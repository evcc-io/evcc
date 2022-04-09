package log

import (
	"strings"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
)

var (
	loggers = map[string]*logger{}
	levels  = map[string]jww.Threshold{}

	loggersMux sync.Mutex

	// OutThreshold is the default console log level
	OutThreshold = jww.LevelError

	// LogThreshold is the default log file level
	LogThreshold = jww.LevelWarn
)

// LogAreaPadding of log areas
var LogAreaPadding = 6

// Loggers invokes callback for each configured logger
func Loggers(cb func(string, *logger)) {
	for name, logger := range loggers {
		cb(name, logger)
	}
}

// LogLevelForArea gets the log level for given log area
func LogLevelForArea(area string) jww.Threshold {
	level, ok := levels[strings.ToLower(area)]
	if !ok {
		level = OutThreshold
	}
	return level
}

// LogLevel sets log level for all loggers
func LogLevel(defaultLevel string, areaLevels map[string]string) {
	// default level
	OutThreshold = LogLevelToThreshold(defaultLevel)
	LogThreshold = OutThreshold

	// area levels
	for area, level := range areaLevels {
		area = strings.ToLower(area)
		levels[area] = LogLevelToThreshold(level)
	}

	Loggers(func(name string, logger *logger) {
		logger.np.SetStdoutThreshold(LogLevelForArea(name))
	})
}

// LogLevelToThreshold converts log level string to a jww Threshold
func LogLevelToThreshold(level string) jww.Threshold {
	switch strings.ToUpper(level) {
	case "FATAL":
		return jww.LevelFatal
	case "ERROR":
		return jww.LevelError
	case "WARN":
		return jww.LevelWarn
	case "INFO":
		return jww.LevelInfo
	case "DEBUG":
		return jww.LevelDebug
	case "TRACE":
		return jww.LevelTrace
	default:
		panic("invalid log level " + level)
	}
}

// var uiChan chan<- Param

// type uiWriter struct {
// 	re    *regexp.Regexp
// 	level string
// }

// func (w *uiWriter) Write(p []byte) (n int, err error) {
// 	// trim level and timestamp
// 	s := string(w.re.ReplaceAll(p, []byte{}))

// 	uiChan <- Param{
// 		Key: w.level,
// 		Val: strings.Trim(strconv.Quote(strings.TrimSpace(s)), "\""),
// 	}

// 	return 0, nil
// }

// // CaptureLogs appends uiWriter to relevant log levels
// func CaptureLogs(c chan<- Param) {
// 	uiChan = c

// 	for _, l := range loggers {
// 		captureLogger("warn", l.Warn)
// 		captureLogger("error", l.Error)
// 		captureLogger("error", l.Fatal)
// 	}
// }

// func captureLogger(level string, func(fmt string, args ...interface{})) {
// 	re, err := regexp.Compile(`^\[[a-zA-Z0-9-]+\s*\] \w+ .{19} `)
// 	if err != nil {
// 		panic(err)
// 	}

// 	ui := uiWriter{
// 		re:    re,
// 		level: level,
// 	}

// 	mw := io.MultiWriter(l.Writer(), &ui)
// 	l.SetOutput(mw)
// }
