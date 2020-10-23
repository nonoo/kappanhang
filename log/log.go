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

func GetCallerFileName(withLine bool) string {
	_, filename, line, _ := runtime.Caller(2)
	extension := filepath.Ext(filename)
	if withLine {
		return fmt.Sprint(filename[filenameTrimChars:len(filename)-len(extension)], "@", line)
	} else {
		return filename[filenameTrimChars : len(filename)-len(extension)]
	}
}

func Printf(a string, b ...interface{}) {
	logger.Infof(GetCallerFileName(false)+": "+a, b...)
}

func Print(a ...interface{}) {
	logger.Info(append([]interface{}{GetCallerFileName(false) + ": "}, a...)...)
}

func Debugf(a string, b ...interface{}) {
	logger.Debugf(GetCallerFileName(true)+": "+a, b...)
}

func Debug(a ...interface{}) {
	logger.Debug(append([]interface{}{GetCallerFileName(true) + ": "}, a...)...)
}

func Errorf(a string, b ...interface{}) {
	logger.Errorf(GetCallerFileName(true)+": "+a, b...)
}

func Error(a ...interface{}) {
	logger.Error(append([]interface{}{GetCallerFileName(true) + ": "}, a...)...)
}

func ErrorC(a ...interface{}) {
	logger.Error(a...)
}

func Fatalf(a string, b ...interface{}) {
	logger.Fatalf(GetCallerFileName(true)+": "+a, b...)
}

func Fatal(a ...interface{}) {
	logger.Fatal(append([]interface{}{GetCallerFileName(true) + ": "}, a...)...)
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
