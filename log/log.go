package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger
var filenameTrimChars int

func getCallerFileName(withLine bool) string {
	_, filename, line, _ := runtime.Caller(2)
	extension := filepath.Ext(filename)
	if withLine {
		return fmt.Sprint(filename[filenameTrimChars:len(filename)-len(extension)], "@", line)
	} else {
		return filename[filenameTrimChars : len(filename)-len(extension)]
	}
}

func Printf(a string, b ...interface{}) {
	logger.Infof(getCallerFileName(false)+": "+a, b...)
}

func Print(a ...interface{}) {
	logger.Info(append([]interface{}{getCallerFileName(false) + ": "}, a...)...)
}

func Debugf(a string, b ...interface{}) {
	logger.Debugf(getCallerFileName(true)+": "+a, b...)
}

func Debug(a ...interface{}) {
	logger.Debug(append([]interface{}{getCallerFileName(true) + ": "}, a...)...)
}

func Errorf(a string, b ...interface{}) {
	logger.Errorf(getCallerFileName(true)+": "+a, b...)
}

func Error(a ...interface{}) {
	logger.Error(append([]interface{}{getCallerFileName(true) + ": "}, a...)...)
}

func Fatalf(a string, b ...interface{}) {
	logger.Fatalf(getCallerFileName(true)+": "+a, b...)
}

func Fatal(a ...interface{}) {
	logger.Fatal(append([]interface{}{getCallerFileName(true) + ": "}, a...)...)
}

func Init() {
	// Example: https://stackoverflow.com/questions/50933936/zap-logger-does-not-print-on-console-rather-print-in-the-log-file/50936341
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	// pe.LevelKey = ""
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	level := zap.DebugLevel

	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	logger = zap.New(core).Sugar()

	var callerFilename string
	_, callerFilename, _, _ = runtime.Caller(1)
	filenameTrimChars = len(filepath.Dir(callerFilename)) + 1
}
