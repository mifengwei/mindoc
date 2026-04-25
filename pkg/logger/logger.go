package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.SugaredLogger

func Init(logPath string, level string, isAsync bool) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var zapLevel zapcore.Level
	switch level {
	case "Debug", "debug":
		zapLevel = zapcore.DebugLevel
	case "Informational", "info":
		zapLevel = zapcore.InfoLevel
	case "Warning", "warn":
		zapLevel = zapcore.WarnLevel
	case "Error", "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.DebugLevel
	}

	cores := []zapcore.Core{}

	// Console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel)
	cores = append(cores, consoleCore)

	// File encoder
	if logPath != "" {
		if _, err := os.Stat(filepath.Dir(logPath)); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(logPath), 0755)
		}

		fileWriter, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
			fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), zapLevel)
			cores = append(cores, fileCore)
		}
	}

	core := zapcore.NewTee(cores...)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	L = zapLogger.Sugar()
}

func ensureInit() {
	if L == nil {
		Init("", "debug", false)
	}
}

func Info(args ...interface{})                  { ensureInit(); L.Info(args...) }
func Infof(template string, args ...interface{}) { ensureInit(); L.Infof(template, args...) }
func Error(args ...interface{})                 { ensureInit(); L.Error(args...) }
func Errorf(template string, args ...interface{}) { ensureInit(); L.Errorf(template, args...) }
func Warn(args ...interface{})                  { ensureInit(); L.Warn(args...) }
func Warnf(template string, args ...interface{}) { ensureInit(); L.Warnf(template, args...) }
func Debug(args ...interface{})                 { ensureInit(); L.Debug(args...) }
func Debugf(template string, args ...interface{}) { ensureInit(); L.Debugf(template, args...) }
func Fatal(args ...interface{})                 { ensureInit(); L.Fatal(args...) }
func Fatalf(template string, args ...interface{}) { ensureInit(); L.Fatalf(template, args...) }
