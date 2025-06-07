package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("CreateLoggerWithPackageName", func(t *testing.T) {
		t.Parallel()

		logger := New("test-package")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("CreateLoggerWithAttributes", func(t *testing.T) {
		t.Parallel()

		attrs := []Attr{
			{Key: "component", Value: "test"},
			{Key: "version", Value: "1.0.0"},
		}

		logger := New("test-package", attrs...)
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("CreateLoggerWithEmptyPackageName", func(t *testing.T) {
		t.Parallel()

		logger := New("")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})
}

func TestSetLogLevel(t *testing.T) {
	t.Parallel()

	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				SetLogLevel(level)
			})
		})
	}
}

func TestInitializeLogging(t *testing.T) {
	t.Run("InitializeLoggingSuccess", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		err := InitializeLogging()
		assert.NoError(t, err)
	})

	t.Run("InitializeLoggingMultipleCalls", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Call multiple times to test sync.Once behavior
		err1 := InitializeLogging()
		err2 := InitializeLogging()
		err3 := InitializeLogging()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
	})

	t.Run("InitializeLoggingWithUnwritableHome", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Create a scenario where we can't write to home directory
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		// Set HOME to a non-existent or unwritable directory
		os.Setenv("HOME", "/nonexistent/unwritable/directory")

		err := InitializeLogging()
		// Should not return an error even if log file creation fails
		assert.NoError(t, err)
	})

	t.Run("InitializeLoggingWithoutHomeDir", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Remove HOME environment variable
		originalHome := os.Getenv("HOME")
		os.Unsetenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			}
		}()

		err := InitializeLogging()
		// Should not return an error even without home directory
		assert.NoError(t, err)
	})
}

func TestGetLogWriter(t *testing.T) {
	t.Run("GetLogWriterWithValidLogFile", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		writer := getLogWriter()
		assert.NotNil(t, writer)

		// Writer should be usable
		n, err := writer.Write([]byte("test log message"))
		assert.NoError(t, err)
		assert.Greater(t, n, 0)
	})

	t.Run("GetLogWriterWithoutLogFile", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Force initialization to fail by setting invalid home
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", "/nonexistent")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		writer := getLogWriter()
		assert.NotNil(t, writer)

		// Should be discard writer, so writing should succeed but do nothing
		n, err := writer.Write([]byte("test log message"))
		assert.NoError(t, err)
		assert.Equal(t, len("test log message"), n)

		// Verify it's actually io.Discard
		assert.Equal(t, io.Discard, writer)
	})
}

func TestLoggerFunctionality(t *testing.T) {
	t.Run("LoggerMethods", func(t *testing.T) {
		t.Parallel()

		logger := New("test")

		// Test that logger methods don't panic
		assert.NotPanics(t, func() {
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")
		})

		assert.NotPanics(t, func() {
			logger.Debug("debug with fields", "key", "value")
			logger.Info("info with fields", "key", "value")
			logger.Warn("warn with fields", "key", "value")
			logger.Error("error with fields", "key", "value")
		})
	})

	t.Run("LoggerWithCustomLevel", func(t *testing.T) {
		t.Parallel()

		// Set debug level
		SetLogLevel(slog.LevelDebug)

		logger := New("test")
		assert.NotPanics(t, func() {
			logger.Debug("debug message should be visible")
		})

		// Set error level
		SetLogLevel(slog.LevelError)

		assert.NotPanics(t, func() {
			logger.Debug("debug message should be filtered")
			logger.Error("error message should be visible")
		})
	})
}

func TestLoggerTimeFormatting(t *testing.T) {
	t.Run("TimeFormatting", func(t *testing.T) {
		t.Parallel()

		// Create a logger and capture its output
		var output strings.Builder

		// Create a custom handler to capture output
		handler := slog.NewTextHandler(&output, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key != slog.TimeKey {
					return a
				}
				return slog.String(
					slog.TimeKey,
					a.Value.Time().Format("2006-01-02T15:04:05.000Z"),
				)
			},
		})

		testLogger := &Logger{slog.New(handler)}
		testLogger.Info("test message")

		result := output.String()
		assert.Contains(t, result, "test message")
		// Should contain the custom time format pattern
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z`, result)
	})
}

func TestLoggerAttributes(t *testing.T) {
	t.Run("LoggerWithAttributes", func(t *testing.T) {
		t.Parallel()

		var output strings.Builder
		handler := slog.NewTextHandler(&output, &slog.HandlerOptions{})

		testLogger := &Logger{
			slog.New(handler.WithAttrs([]slog.Attr{
				{Key: "pkg", Value: slog.StringValue("test-package")},
				{Key: "component", Value: slog.StringValue("test-component")},
			})),
		}

		testLogger.Info("test message")

		result := output.String()
		assert.Contains(t, result, "test message")
		assert.Contains(t, result, "pkg=test-package")
		assert.Contains(t, result, "component=test-component")
	})
}

func TestConcurrentLoggerUsage(t *testing.T) {
	t.Run("ConcurrentLogging", func(t *testing.T) {
		t.Parallel()

		logger := New("concurrent-test")
		const numGoroutines = 10
		const messagesPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					logger.Info("concurrent message", "goroutine", id, "message", j)
				}
			}(i)
		}

		// Should not deadlock or panic
		assert.NotPanics(t, func() {
			wg.Wait()
		})
	})
}

func TestErrorPathsInLogFileCreation(t *testing.T) {
	t.Run("ErrorCreatingLogDirectory", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Create a temporary directory that we'll make read-only
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0755)
		require.NoError(t, err)

		// Make it read-only
		err = os.Chmod(readOnlyDir, 0444)
		require.NoError(t, err)

		// Set HOME to a path inside the read-only directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", readOnlyDir)
		defer func() {
			// Restore permissions to clean up
			os.Chmod(readOnlyDir, 0755)
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		// This should not fail even if directory creation fails
		err = InitializeLogging()
		assert.NoError(t, err)

		// Writer should be discard
		writer := getLogWriter()
		assert.Equal(t, io.Discard, writer)
	})

	t.Run("ErrorOpeningLogFile", func(t *testing.T) {
		// Reset the global state for this test
		resetLoggerState()

		// Create a directory but with no write permissions
		tempDir := t.TempDir()
		logDir := filepath.Join(tempDir, ".rudder")
		err := os.MkdirAll(logDir, 0755)
		require.NoError(t, err)

		// Create a file where we want the log to be, but make it unwritable
		logFile := filepath.Join(logDir, "cli.log")
		err = os.WriteFile(logFile, []byte("existing content"), 0444)
		require.NoError(t, err)

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer func() {
			// Restore permissions to clean up
			os.Chmod(logFile, 0644)
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		// This should not fail even if log file opening fails
		err = InitializeLogging()
		assert.NoError(t, err)

		// Writer should be discard since file opening failed
		writer := getLogWriter()
		assert.Equal(t, io.Discard, writer)
	})
}

// Helper function to reset logger global state for testing
func resetLoggerState() {
	// Close existing log file if open
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	// Reset the sync.Once
	initOnce = sync.Once{}
	initErr = nil
}
