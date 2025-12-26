package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	t.Run("development environment", func(t *testing.T) {
		logger := New("debug", "development")
		assert.NotNil(t, logger)

		logger.Debug("test debug")
		logger.Info("test info")
		logger.Warn("test warn")
		logger.Error("test error")
	})

	t.Run("production environment", func(t *testing.T) {
		logger := New("info", "production")
		assert.NotNil(t, logger)

		logger.Info("test info")
		logger.Warn("test warn")
	})

	t.Run("invalid log level", func(t *testing.T) {
		logger := New("invalid", "development")
		assert.NotNil(t, logger)

		logger.Info("test info")
	})
}

func TestNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("debug", &buf)

	logger.Info("test message")

	assert.Contains(t, buf.String(), "test message")
	assert.Contains(t, buf.String(), "level")
}

func TestLoggerMethods(t *testing.T) {
	logger := New("debug", "test")

	logger.Debug("debug message")
	logger.Debugf("debug format: %s", "test")
	logger.Info("info message")
	logger.Infof("info format: %s", "test")
	logger.Warn("warn message")
	logger.Warnf("warn format: %s", "test")
	logger.Error("error message")
	logger.Errorf("error format: %s", "test")
}

func TestLoggerWithFields(t *testing.T) {
	logger := New("info", "test")

	withField := logger.WithField("component", "test_component")
	assert.NotNil(t, withField)

	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}
	withFields := logger.WithFields(fields)
	assert.NotNil(t, withFields)
}

func TestFormatError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FormatError(nil)
		assert.Equal(t, "", result)
	})

	t.Run("non-nil error", func(t *testing.T) {
		err := errors.New("test error")
		result := FormatError(err)
		assert.Equal(t, "test error", result)
	})
}

func TestWithContext(t *testing.T) {
	logger := New("info", "test")

	contextLogger := WithContext(logger, "test_context")
	assert.NotNil(t, contextLogger)
}

func TestSetLevel(t *testing.T) {
	logger := New("info", "test")

	err := SetLevel(logger, "debug")
	assert.NoError(t, err)

	err = SetLevel(logger, "invalid")
	assert.Error(t, err)
}

func TestIsDebugEnabled(t *testing.T) {
	t.Run("debug level enabled", func(t *testing.T) {
		logger := New("debug", "test")
		assert.True(t, IsDebugEnabled(logger))
	})

	t.Run("info level disabled for debug", func(t *testing.T) {
		logger := New("info", "test")
		assert.False(t, IsDebugEnabled(logger))
	})
}

func TestStringToLevel(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"debug", "debug", "debug"},
		{"info", "info", "info"},
		{"warn", "warn", "warning"},
		{"warning", "warning", "warning"},
		{"error", "error", "error"},
		{"fatal", "fatal", "fatal"},
		{"panic", "panic", "panic"},
		{"invalid", "invalid", "info"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			level := StringToLevel(tc.input)
			assert.Equal(t, tc.expected, level.String())
		})
	}
}

func TestLogrusLogger_Interface(t *testing.T) {
	var _ Logger = (*logrusLogger)(nil)
}

func TestLogger_WithField_Chaining(t *testing.T) {
	logger := New("info", "test")

	logger1 := logger.WithField("component", "test")
	logger2 := logger1.WithField("version", "1.0")
	logger3 := logger2.WithField("environment", "testing")

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)
	assert.NotNil(t, logger3)

	logger3.Info("test message with fields")
}

func TestLogger_WithFields_Chaining(t *testing.T) {
	logger := New("info", "test")

	fields1 := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}

	fields2 := map[string]interface{}{
		"field3": true,
		"field4": 45.6,
	}

	logger1 := logger.WithFields(fields1)
	logger2 := logger1.WithFields(fields2)

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)

	logger2.Info("test message with multiple fields")
}

func TestNew_InvalidEnvironment(t *testing.T) {
	logger := New("info", "invalid_env")

	assert.NotNil(t, logger)
	logger.Info("test message")
}

func TestSetLevel_OnNonLogrusLogger(t *testing.T) {
	type mockLogger struct {
		Logger
	}

	mock := &mockLogger{}

	err := SetLevel(mock, "debug")

	assert.NoError(t, err)
}

func TestIsDebugEnabled_OnNonLogrusLogger(t *testing.T) {
	type mockLogger struct {
		Logger
	}

	mock := &mockLogger{}

	enabled := IsDebugEnabled(mock)
	assert.False(t, enabled)
}

func TestLogger_ProductionFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)

	logger.Info("test message")

	output := buf.String()

	assert.True(t, strings.Contains(output, "\"level\""), "Output should contain JSON field 'level'")
	assert.True(t, strings.Contains(output, "\"msg\""), "Output should contain JSON field 'msg'")
	assert.True(t, strings.Contains(output, "test message"), "Output should contain the message")
}

func TestLogger_WithField_ReturnsNewLogger(t *testing.T) {
	logger := New("info", "test")

	loggerWithField := logger.WithField("component", "test_component")

	assert.NotNil(t, loggerWithField)
	assert.NotEqual(t, logger, loggerWithField)

	assert.NotPanics(t, func() {
		loggerWithField.Info("test message")
	})
}

func TestLogger_WithFields_ReturnsNewLogger(t *testing.T) {
	logger := New("info", "test")

	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}

	loggerWithFields := logger.WithFields(fields)

	assert.NotNil(t, loggerWithFields)
	assert.NotEqual(t, logger, loggerWithFields)

	assert.NotPanics(t, func() {
		loggerWithFields.Info("test message with fields")
	})
}

func TestNew_InvalidLevel_DefaultsToInfo(t *testing.T) {
	logger := New("invalid-level", "development")

	assert.NotNil(t, logger)

	assert.NotPanics(t, func() {
		logger.Info("test info message")
		logger.Debug("this might not appear if level is info")
	})
}

func TestNewWithWriter_ProductionFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)

	logger.Info("test message", "additional", "data")

	output := buf.String()

	assert.True(t, strings.Contains(output, "\"level\""), "Should contain level field")
	assert.True(t, strings.Contains(output, "\"msg\""), "Should contain msg field")
	assert.True(t, strings.Contains(output, "test message"), "Should contain message")
}

func TestIsDebugEnabled_WithDebugLevel(t *testing.T) {
	logger := New("debug", "test")

	enabled := IsDebugEnabled(logger)

	assert.True(t, enabled, "Debug should be enabled with debug level")
}

func TestIsDebugEnabled_WithInfoLevel(t *testing.T) {
	logger := New("info", "test")

	enabled := IsDebugEnabled(logger)

	assert.False(t, enabled, "Debug should not be enabled with info level")
}

func TestStringToLevel_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"DEBUG", "debug"},
		{"Debug", "debug"},
		{"INFO", "info"},
		{"Info", "info"},
		{"WARN", "warning"},
		{"WARNING", "warning"},
		{"ERROR", "error"},
		{"FATAL", "fatal"},
		{"PANIC", "panic"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			level := StringToLevel(tc.input)
			assert.Equal(t, tc.expected, level.String())
		})
	}
}

func TestSetLevel_NonLogrusLogger(t *testing.T) {
	type customLogger struct{ Logger }
	logger := &customLogger{}

	err := SetLevel(logger, "debug")
	assert.NoError(t, err)
}

func TestIsDebugEnabled_NonLogrusLogger(t *testing.T) {
	type customLogger struct{ Logger }
	logger := &customLogger{}

	enabled := IsDebugEnabled(logger)
	assert.False(t, enabled)
}

func TestNew_EmptyEnvironment(t *testing.T) {
	logger := New("info", "")
	assert.NotNil(t, logger)

	logger.Info("test with empty env")
}

func TestFormatError_Nil(t *testing.T) {
	result := FormatError(nil)
	assert.Equal(t, "", result)
}

func TestLogger_WithContext(t *testing.T) {
	logger := New("info", "test")

	contextLogger := WithContext(logger, "test-context")
	assert.NotNil(t, contextLogger)

	contextLogger.Info("test with context")
}

func TestSetLevel_InvalidLevel(t *testing.T) {
	logger := New("info", "test")

	err := SetLevel(logger, "invalid-level")
	assert.Error(t, err)
}

func TestStringToLevel_Unknown(t *testing.T) {
	level := StringToLevel("unknown")
	assert.Equal(t, logrus.InfoLevel, level)
}

func TestNewWithWriter_CustomWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("debug", &buf)

	logger.Debug("test debug")
	logger.Info("test info")

	output := buf.String()
	assert.Contains(t, output, "test debug")
	assert.Contains(t, output, "test info")
}

func TestLogger_Fatal_Methods(t *testing.T) {
	t.Run("Fatal in development", func(t *testing.T) {
		var buf bytes.Buffer

		loggerObj := logrus.New()
		loggerObj.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
		loggerObj.SetLevel(logrus.InfoLevel)
		loggerObj.SetOutput(&buf)
		loggerObj.ExitFunc = func(code int) {
		}

		logger := &logrusLogger{
			entry: logrus.NewEntry(loggerObj),
		}

		logger.Fatal("test fatal")

		output := buf.String()
		assert.Contains(t, output, "test fatal")
		assert.Contains(t, output, "FATA")
	})

	t.Run("Fatalf in production", func(t *testing.T) {
		var buf bytes.Buffer

		loggerObj := logrus.New()
		loggerObj.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
		loggerObj.SetLevel(logrus.InfoLevel)
		loggerObj.SetOutput(&buf)
		loggerObj.ExitFunc = func(code int) {
		}

		logger := &logrusLogger{
			entry: logrus.NewEntry(loggerObj),
		}

		logger.Fatalf("test fatal: %s", "error")

		output := buf.String()
		assert.Contains(t, output, "test fatal: error")
		assert.Contains(t, output, "\"level\":\"fatal\"")
	})
}

func TestNewWithWriter_InvalidLogLevel(t *testing.T) {
	var buf bytes.Buffer

	logger := NewWithWriter("invalid-level", &buf)

	assert.NotNil(t, logger)

	logger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
}

func TestNewWithWriter_ValidLogLevel(t *testing.T) {
	var buf bytes.Buffer

	logger := NewWithWriter("debug", &buf)

	assert.NotNil(t, logger)

	logger.Debug("debug message")
	logger.Info("info message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
}
