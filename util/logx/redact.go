package logx

import (
	"fmt"
	"reflect"

	"github.com/evcc-io/evcc/util/logx/redact"
)

type redactLogger struct {
	Logger
	*redact.Redactor
}

func Redact(log Logger, items ...string) Logger {
	wrapper := &redactLogger{
		Logger:   log,
		Redactor: new(redact.Redactor),
	}

	wrapper.Redactor.Redact(items...)

	return wrapper
}

func (l *redactLogger) Log(keyvals ...interface{}) error {
	kv := make([]interface{}, 0, len(keyvals))

	for _, v := range keyvals {
		kv = append(kv, l.encode(v))
	}

	return l.Logger.Log(kv...)
}

func (l *redactLogger) encode(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return l.Redactor.Safe(v)
	case error:
		val, _ := safeError(v)
		return l.Redactor.Safe(val)
	case fmt.Stringer:
		val, _ := safeString(v)
		return l.Redactor.Safe(val)
	default:
		return value
	}
}

// https://github.com/go-logfmt/logfmt/blob/master/encode.go
func safeError(err error) (s string, ok bool) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if v := reflect.ValueOf(err); v.Kind() == reflect.Ptr && v.IsNil() {
				s, ok = "null", false
			} else {
				s, ok = fmt.Sprintf("PANIC:%v", panicVal), false
			}
		}
	}()
	s, ok = err.Error(), true
	return
}

// https://github.com/go-logfmt/logfmt/blob/master/encode.go
func safeString(str fmt.Stringer) (s string, ok bool) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if v := reflect.ValueOf(str); v.Kind() == reflect.Ptr && v.IsNil() {
				s, ok = "null", false
			} else {
				s, ok = fmt.Sprintf("PANIC:%v", panicVal), true
			}
		}
	}()
	s, ok = str.String(), true
	return
}
