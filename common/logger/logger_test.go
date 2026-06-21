package logger

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogger_ReturnsSameInstance(t *testing.T) {
	first := GetLogger()
	second := GetLogger()

	require.NotNil(t, first)
	assert.Same(t, first, second)
	assert.Same(t, first, logger)
}

func TestGetLogger_ConfiguresExpectedFormatter(t *testing.T) {
	instance := GetLogger()

	formatter, ok := instance.Formatter.(*logrus.TextFormatter)
	require.True(t, ok)
	assert.True(t, formatter.DisableColors)
	assert.True(t, formatter.FullTimestamp)
}

func TestWithHelpers_ReturnEntriesBoundToDefaultLogger(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "req-1")
	moment := time.Unix(1717392000, 0)

	entryWithField := WithField("scope", "logger")
	entryWithFields := WithFields(logrus.Fields{"feature": "shared", "status": "ok"})
	err := assert.AnError
	entryWithError := WithError(err)
	entryWithContext := WithContext(ctx)
	entryWithTime := WithTime(moment)

	require.NotNil(t, entryWithField)
	require.NotNil(t, entryWithFields)
	require.NotNil(t, entryWithError)
	require.NotNil(t, entryWithContext)
	require.NotNil(t, entryWithTime)

	assert.Same(t, GetLogger(), entryWithField.Logger)
	assert.Same(t, GetLogger(), entryWithFields.Logger)
	assert.Same(t, GetLogger(), entryWithError.Logger)
	assert.Same(t, GetLogger(), entryWithContext.Logger)
	assert.Same(t, GetLogger(), entryWithTime.Logger)
	assert.Equal(t, "logger", entryWithField.Data["scope"])
	assert.Equal(t, "shared", entryWithFields.Data["feature"])
	assert.Equal(t, "ok", entryWithFields.Data["status"])
	assert.Equal(t, err, entryWithError.Data[logrus.ErrorKey])
	assert.Equal(t, ctx, entryWithContext.Context)
	assert.True(t, entryWithTime.Time.Equal(moment))
}

func TestPackageHelpers_DoNotPanic(t *testing.T) {
	instance := GetLogger()
	originalOut := instance.Out
	instance.SetOutput(io.Discard)
	defer instance.SetOutput(originalOut)

	assert.NotPanics(t, func() {
		Log(logrus.InfoLevel, "plain")
		Trace("trace")
		Debug("debug")
		Info("info")
		Print("print")
		Warn("warn")
		Error("error")

		Logf(logrus.InfoLevel, "value=%d", 1)
		Tracef("value=%d", 2)
		Debugf("value=%d", 3)
		Infof("value=%d", 4)
		Printf("value=%d", 5)
		Warnf("value=%d", 6)
		Errorf("value=%d", 7)

		Logln(logrus.InfoLevel, "line")
		Traceln("trace line")
		Debugln("debug line")
		Infoln("info line")
		Println("print line")
		Warnln("warn line")
		Errorln("error line")
	})
}
