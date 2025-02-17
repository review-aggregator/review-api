package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Examples of usage

// logger.Info("This is info log", "Here args are separated by spaces")
// logger.Infow("This is info log with key-value pairs", "uid", 123456)
// logger.Error("This is error log", err)
// logger.Fatal("This is fatal log", err)
// logger.Panic("This is panic log", err)

type logger struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

func (l *logger) Sync(args ...interface{}) {
	l.logger.Sync()
	l.sugaredLogger.Sync()
}

func (l *logger) Debug(args ...interface{}) {
	l.sugaredLogger.Debugln(args...)
}

func (l *logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Debugw(msg, keysAndValues...)
}

func (l *logger) Info(args ...interface{}) {
	l.sugaredLogger.Infoln(args...)
}

func (l *logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Infow(msg, keysAndValues...)
}

func (l *logger) Warn(args ...interface{}) {
	l.sugaredLogger.Warnln(args...)
}

func (l *logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Warnw(msg, keysAndValues...)
}

func (l *logger) Error(msg string, err error) {
	l.logger.Error(msg, zap.Error(err))
}

func (l *logger) Fatal(msg string, err error) {
	if err != nil {
		l.logger.Fatal(msg, zap.Error(err))
		return
	}
	l.logger.Fatal(msg)
}

func (l *logger) Panic(msg string, err error) {
	if err != nil {
		l.logger.Panic(msg, zap.Error(err))
		return
	}
	l.logger.Panic(msg)
}

func PrettyStruct(data interface{}) string {
	bytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Sprintf("Error marshaling struct: %v", err)
	}
	return string(bytes)
}

func CreateLogger() *logger {
	stdout := zapcore.AddSync(os.Stdout)

	level := zap.NewAtomicLevelAt(zap.InfoLevel)

	productionCfg := zap.NewProductionEncoderConfig()
	productionCfg.TimeKey = "timestamp"
	productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	developmentCfg := zap.NewDevelopmentEncoderConfig()
	developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
	jsonEncoder := zapcore.NewJSONEncoder(productionCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(jsonEncoder, stdout, level),
	)

	newLogger := logger{}
	newLogger.logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	newLogger.sugaredLogger = newLogger.logger.Sugar()
	return &newLogger
}
