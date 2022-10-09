package db

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/util"
	"gorm.io/gorm/logger"
)

type Logger struct {
	log *util.Logger
}

func (l *Logger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l *Logger) Info(_ context.Context, msg string, val ...interface{}) {
	l.log.INFO.Printf(msg, val...)
}

func (l *Logger) Warn(_ context.Context, msg string, val ...interface{}) {
	l.log.WARN.Printf(msg, val...)
}

func (l *Logger) Error(_ context.Context, msg string, val ...interface{}) {
	l.log.ERROR.Printf(msg, val...)
}

func (l *Logger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	l.log.TRACE.Println(sql, rows, err)
}
