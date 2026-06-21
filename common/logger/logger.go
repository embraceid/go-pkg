package logger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = newLogger()
}

func newLogger() *logrus.Logger {
	instance := logrus.New()
	instance.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	return instance
}

func GetLogger() *logrus.Logger {
	if logger == nil {
		logger = newLogger()
	}
	return logger
}

func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return GetLogger().WithError(err)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return GetLogger().WithContext(ctx)
}

func WithTime(t time.Time) *logrus.Entry {
	return GetLogger().WithTime(t)
}

func Log(level logrus.Level, args ...interface{}) {
	GetLogger().Log(level, args...)
}

func Trace(args ...interface{}) {
	GetLogger().Trace(args...)
}

func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Print(args ...interface{}) {
	GetLogger().Print(args...)
}

func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

func Logf(level logrus.Level, format string, args ...interface{}) {
	GetLogger().Logf(level, format, args...)
}

func Tracef(format string, args ...interface{}) {
	GetLogger().Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	GetLogger().Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

func Logln(level logrus.Level, args ...interface{}) {
	GetLogger().Logln(level, args...)
}

func Traceln(args ...interface{}) {
	GetLogger().Traceln(args...)
}

func Debugln(args ...interface{}) {
	GetLogger().Debugln(args...)
}

func Infoln(args ...interface{}) {
	GetLogger().Infoln(args...)
}

func Println(args ...interface{}) {
	GetLogger().Println(args...)
}

func Warnln(args ...interface{}) {
	GetLogger().Warnln(args...)
}

func Errorln(args ...interface{}) {
	GetLogger().Errorln(args...)
}
