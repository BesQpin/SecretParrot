package logging

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
	debug bool
}

func New() *Logger {
	debug := os.Getenv("DEBUG") != ""
	l := log.New(os.Stdout, "[secret-parrot] ", log.LstdFlags|log.Lmicroseconds)
	return &Logger{l, debug}
}

func (l *Logger) Printf(format string, v ...any) { l.Logger.Printf(format, v...) }
func (l *Logger) Fatalf(format string, v ...any) { l.Logger.Fatalf(format, v...) }
func (l *Logger) Debugf(format string, v ...any) {
	if l.debug {
		l.Logger.Printf("[DEBUG] "+format, v...)
	}
}
