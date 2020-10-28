package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	logger            *zap.SugaredLogger
	filenameTrimChars int
}

var log logger

func (l *logger) GetCallerFileName(withLine bool) string {
	_, filename, line, _ := runtime.Caller(2)
	extension := filepath.Ext(filename)
	if withLine {
		return fmt.Sprint(filename[l.filenameTrimChars:len(filename)-len(extension)], "@", line)
	} else {
		return filename[l.filenameTrimChars : len(filename)-len(extension)]
	}
}

func (l *logger) printLineClear() {
	fmt.Printf("%c[2K", 27)
}

func (l *logger) Printf(a string, b ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Infof(l.GetCallerFileName(false)+": "+a, b...)
}

func (l *logger) Print(a ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Info(append([]interface{}{l.GetCallerFileName(false) + ": "}, a...)...)
}

func (l *logger) Debugf(a string, b ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Debugf(l.GetCallerFileName(true)+": "+a, b...)
}

func (l *logger) Debug(a ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Debug(append([]interface{}{l.GetCallerFileName(true) + ": "}, a...)...)
}

func (l *logger) Errorf(a string, b ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Errorf(l.GetCallerFileName(true)+": "+a, b...)
}

func (l *logger) Error(a ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Error(append([]interface{}{l.GetCallerFileName(true) + ": "}, a...)...)
}

func (l *logger) ErrorC(a ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Error(a...)
}

func (l *logger) Fatalf(a string, b ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Fatalf(l.GetCallerFileName(true)+": "+a, b...)
}

func (l *logger) Fatal(a ...interface{}) {
	if ctrl != nil {
		l.printLineClear()
	}
	l.logger.Fatal(append([]interface{}{l.GetCallerFileName(true) + ": "}, a...)...)
}

func (l *logger) Init() {
	// Example: https://stackoverflow.com/questions/50933936/zap-logger-does-not-print-on-console-rather-print-in-the-log-file/50936341
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	// pe.LevelKey = ""
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	var level zapcore.Level
	if verboseLog {
		level = zap.DebugLevel
	} else {
		level = zap.InfoLevel
	}

	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	l.logger = zap.New(core).Sugar()

	var callerFilename string
	_, callerFilename, _, _ = runtime.Caller(1)
	l.filenameTrimChars = len(filepath.Dir(callerFilename)) + 1
}
