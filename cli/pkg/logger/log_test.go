package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Creation(t *testing.T) {
	t.Parallel()

	t.Run("LoggerCreationWithNoAttributes", func(t *testing.T) {
		t.Parallel()

		logger := New("test-pkg")
		assert.NotNil(t, logger)

		// Should work without panicking
		assert.NotPanics(t, func() {
			logger.Info("test message")
		})
	})

	t.Run("LoggerCreationWithAttributes", func(t *testing.T) {
		t.Parallel()

		logger := New("test-pkg", Attr{Key: "env", Value: "test"}, Attr{Key: "version", Value: "1.0.0"})
		assert.NotNil(t, logger)

		// Should work without panicking
		assert.NotPanics(t, func() {
			logger.Info("test message with attributes")
		})
	})

	t.Run("MultipleLoggerInstances", func(t *testing.T) {
		t.Parallel()

		logger1 := New("pkg1")
		logger2 := New("pkg2")

		assert.NotNil(t, logger1)
		assert.NotNil(t, logger2)

		// Both should work without panicking
		assert.NotPanics(t, func() {
			logger1.Info("message from logger1")
			logger2.Info("message from logger2")
		})
	})
}

func TestLogger_LoggingMethods(t *testing.T) {
	t.Parallel()

	logger := New("test-pkg")
	require.NotNil(t, logger)

	t.Run("AllLogLevels", func(t *testing.T) {
		t.Parallel()

		// All logging methods should work without panicking
		assert.NotPanics(t, func() {
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")
		})
	})

	t.Run("LoggingWithContext", func(t *testing.T) {
		t.Parallel()

		// Logging with additional context should work
		assert.NotPanics(t, func() {
			logger.Info("message with context", "key1", "value1", "key2", "value2")
			logger.Error("error with context", "error", "some error", "code", 500)
		})
	})
}

func TestSetLogLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		level slog.Level
	}{
		{
			name:  "DebugLevel",
			level: slog.LevelDebug,
		},
		{
			name:  "InfoLevel",
			level: slog.LevelInfo,
		},
		{
			name:  "WarnLevel",
			level: slog.LevelWarn,
		},
		{
			name:  "ErrorLevel",
			level: slog.LevelError,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				SetLogLevel(c.level)
			})
		})
	}
}

func TestInitializeLogging(t *testing.T) {
	t.Parallel()

	t.Run("InitializeLoggingIsSafe", func(t *testing.T) {
		t.Parallel()

		// InitializeLogging should never return an error with our current implementation
		err := InitializeLogging()
		assert.NoError(t, err)

		// Calling it multiple times should be safe
		err = InitializeLogging()
		assert.NoError(t, err)
	})

	t.Run("LoggerWorksAfterInitialization", func(t *testing.T) {
		t.Parallel()

		// Initialize logging explicitly
		err := InitializeLogging()
		assert.NoError(t, err)

		// Create a logger after initialization
		logger := New("post-init-test")
		require.NotNil(t, logger)

		// Logger should work
		assert.NotPanics(t, func() {
			logger.Info("message after explicit initialization")
		})
	})
}
