/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package zlog

import (
	"chain/lib/queuestore"
	"chain/utils"
	"fmt"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewStdoutLogger() *Logger {

	return &Logger{
		s: NewZapSugaredLogger(
			[]OutputSpec{
				{
					Target: "stdout",
					Format: "console",
				},
			},
		),
	}
}

// NewDefaultAppLog return default zlog of current process which logpath depends on the process name.
func NewDefaultAppLog() *Logger {
	pi := utils.GetCurrentProcessInfo()
	procName := strings.TrimSuffix(filepath.Base(pi.ImagePath), filepath.Ext(pi.ImagePath))
	procName = strings.ToLower(procName)

	//extract appName from commandline
	_, appName, _ := utils.ParseCmdLineString("")
	if appName != "" {
		appName = "_" + appName
	}

	if strings.Contains(procName, "nodeproxy") {
		procName = "nodeproxy"
	} else {
		if strings.Contains(procName, "nodemaster") {
			procName = "nodemaster"
		}
	}

	return NewLogger(fmt.Sprintf("%s%s", procName, appName))
}

// NewLogger creates a zlog that delegates to the zap.SugaredLogger.
func NewLogger(appName string) *Logger {
	return &Logger{
		s: NewZapSugaredLogger(
			[]OutputSpec{
				{
					Target: filepath.Join(utils.GetLogPath(appName), appName+".log"),
					Format: "json",
				},
				{
					Target: "stdout",
					Format: "console",
				},
			}),
	}
}

func NewCometbftNodeLogger(nodeID, logPath string, store queuestore.QueueStore) *Logger {
	return &Logger{
		s: NewStreamSugaredLogger(nodeID, logPath, store),
	}
}

// A Logger is an adapter around a zap.SugaredLogger that provides
// structured logging capabilities while preserving much of the legacy logging
// behavior.
//
// The most significant difference between the Logger and the
// zap.SugaredLogger is that methods without a formatting suffix (f or w) build
// the zlog entry message with fmt.Sprintln instead of fmt.Sprint. Without this
// change, arguments are not separated by spaces.
type Logger struct {
	s *zap.SugaredLogger
}

func (f *Logger) Write(p []byte) (n int, err error) {
	f.s.Info(string(p))
	return len(p), nil
}

var l = NewDefaultAppLog()

func (f *Logger) AddWriter(core zapcore.Core) error {

	if core == nil {
		return nil
	}

	combinedCore := zapcore.NewTee(f.s.Desugar().Core(), core)

	zapLogger := zap.New(
		combinedCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	f.s = zapLogger.Sugar()

	return nil
}

func (f *Logger) AddOutput(appName string, logBroker *queuestore.Broker, format string, level zapcore.Level) error {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")

	var encoder zapcore.Encoder
	if format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	brokerCore := &brokerCore{
		broker: logBroker,
		name:   appName,
		enc:    encoder,
		level:  level,
	}

	return f.AddWriter(brokerCore)
}

func Info(v ...interface{})                  { l.Info(v...) }
func Infof(format string, v ...interface{})  { l.Infof(format, v...) }
func Warn(v ...interface{})                  { l.Warn(v...) }
func Warnf(format string, v ...interface{})  { l.Warnf(format, v...) }
func Error(v ...interface{})                 { l.Error(v...) }
func Errorf(format string, v ...interface{}) { l.Errorf(format, v...) }
func Println(v ...interface{})               { l.Println(v...) }
func Printf(format string, v ...interface{}) { l.Printf(format, v...) }
func Panic(v ...interface{})                 { l.Panic(v...) }
func Panicf(format string, v ...interface{}) { l.Panicf(format, v...) }
func Fatal(v ...interface{})                 { l.Fatal(v...) }
func Fatalf(format string, v ...interface{}) { l.Fatalf(format, v...) }

func (f *Logger) Println(v ...any)                              { f.s.Info(v...) }
func (f *Logger) Printf(format string, v ...any)                { f.s.Infof(format, v...) }
func (f *Logger) DPanic(args ...interface{})                    { f.s.DPanicf(formatArgs(args)) }
func (f *Logger) DPanicf(template string, args ...interface{})  { f.s.DPanicf(template, args...) }
func (f *Logger) DPanicw(msg string, kvPairs ...interface{})    { f.s.DPanicw(msg, kvPairs...) }
func (f *Logger) Debug(args ...interface{})                     { f.s.Debugf(formatArgs(args)) }
func (f *Logger) Debugf(template string, args ...interface{})   { f.s.Debugf(template, args...) }
func (f *Logger) Debugw(msg string, kvPairs ...interface{})     { f.s.Debugw(msg, kvPairs...) }
func (f *Logger) Error(args ...interface{})                     { f.s.Errorf(formatArgs(args)) }
func (f *Logger) Errorf(template string, args ...interface{})   { f.s.Errorf(template, args...) }
func (f *Logger) Errorw(msg string, kvPairs ...interface{})     { f.s.Errorw(msg, kvPairs...) }
func (f *Logger) Fatal(args ...interface{})                     { f.s.Fatalf(formatArgs(args)) }
func (f *Logger) Fatalf(template string, args ...interface{})   { f.s.Fatalf(template, args...) }
func (f *Logger) Fatalw(msg string, kvPairs ...interface{})     { f.s.Fatalw(msg, kvPairs...) }
func (f *Logger) Info(args ...interface{})                      { f.s.Infof(formatArgs(args)) }
func (f *Logger) Infof(template string, args ...interface{})    { f.s.Infof(template, args...) }
func (f *Logger) Infow(msg string, kvPairs ...interface{})      { f.s.Infow(msg, kvPairs...) }
func (f *Logger) Panic(args ...interface{})                     { f.s.Panicf(formatArgs(args)) }
func (f *Logger) Panicf(template string, args ...interface{})   { f.s.Panicf(template, args...) }
func (f *Logger) Panicw(msg string, kvPairs ...interface{})     { f.s.Panicw(msg, kvPairs...) }
func (f *Logger) Warn(args ...interface{})                      { f.s.Warnf(formatArgs(args)) }
func (f *Logger) Warnf(template string, args ...interface{})    { f.s.Warnf(template, args...) }
func (f *Logger) Warnw(msg string, kvPairs ...interface{})      { f.s.Warnw(msg, kvPairs...) }
func (f *Logger) Warning(args ...interface{})                   { f.s.Warnf(formatArgs(args)) }
func (f *Logger) Warningf(template string, args ...interface{}) { f.s.Warnf(template, args...) }

// for backwards compatibility
func (f *Logger) Critical(args ...interface{})                   { f.s.Errorf(formatArgs(args)) }
func (f *Logger) Criticalf(template string, args ...interface{}) { f.s.Errorf(template, args...) }
func (f *Logger) Notice(args ...interface{})                     { f.s.Infof(formatArgs(args)) }
func (f *Logger) Noticef(template string, args ...interface{})   { f.s.Infof(template, args...) }

func (f *Logger) With(args ...interface{}) *Logger {
	return &Logger{s: f.s.With(args...)}
}

func formatArgs(args []interface{}) string { return strings.TrimSuffix(fmt.Sprintln(args...), "\n") }
