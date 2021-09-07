package api

type Logger interface {
	Tracef(format string, v ...interface{})
	Traceln(v ...interface{})
	Debugf(format string, v ...interface{})
	Debugln(v ...interface{})
	Infof(format string, v ...interface{})
	Infoln(v ...interface{})
	Warnf(format string, v ...interface{})
	Warnln(v ...interface{})
	Errorf(format string, v ...interface{})
	Errorln(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
}
