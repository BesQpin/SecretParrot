package logging

import (
	"log"
	"os"
)

type Logger struct{ *log.Logger }

func New() *Logger {
	l := log.New(os.Stdout, "[secret-parrot] ", log.LstdFlags|log.Lmicroseconds)
	return &Logger{l}
}

func (l *Logger) Printf(format string, v ...any) { l.Logger.Printf(format, v...) }
func (l *Logger) Fatalf(format string, v ...any) { l.Logger.Fatalf(format, v...) }
