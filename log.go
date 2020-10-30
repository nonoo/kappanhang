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
	}
	return filename[l.filenameTrimChars : len(filename)-len(extension)]
}

func (l *logger) Print(a ...interface{}) {
	if statusLog.isRealtime() {
		statusLog.mutex.Lock()
		statusLog.clearInternal()
		defer func() {
			statusLog.mutex.Unlock()
			statusLog.print()
		}()
	}
	l.logger.Info(append([]interface{}{l.GetCallerFileName(false) + ": "}, a...)...)
}

func (l *logger) PrintStatusLog(a ...interface{}) {
	l.logger.Info(append([]interface{}{l.GetCallerFileName(false) + ": "}, a...)...)
}

func (l *logger) Debug(a ...interface{}) {
	if statusLog.isRealtime() {
		statusLog.mutex.Lock()
		statusLog.clearInternal()
		defer func() {
			statusLog.mutex.Unlock()
			statusLog.print()
		}()
	}
	l.logger.Debug(append([]interface{}{l.GetCallerFileName(true) + ": "}, a...)...)
}

func (l *logger) Error(a ...interface{}) {
	if statusLog.isRealtime() {
		statusLog.mutex.Lock()
		statusLog.clearInternal()
		defer func() {
			statusLog.mutex.Unlock()
			statusLog.print()
		}()
	}
	l.logger.Error(append([]interface{}{l.GetCallerFileName(true) + ": "}, a...)...)
}

func (l *logger) ErrorC(a ...interface{}) {
	if statusLog.isRealtime() {
		statusLog.mutex.Lock()
		statusLog.clearInternal()
		defer func() {
			statusLog.mutex.Unlock()
			statusLog.print()
		}()
	}
	l.logger.Error(a...)
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
