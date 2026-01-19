package writer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubFormatter implements formatter.Formatter for testing.
type stubFormatter struct {
	exts []string
	out  []byte
	err  error
}

func (s stubFormatter) Format(data any) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func (s stubFormatter) Extension() []string { return s.exts }

func TestWrite(t *testing.T) {
	t.Run("creates file exclusive and fails on second write", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir = t.TempDir()
			ctx    = context.Background()
		)

		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("test-bytes")})

		entities := []FormattableEntity{{
			Content:      map[string]any{"k": "v"},
			RelativePath: "out.yaml",
		}}

		err := Write(ctx, tmpDir, formatters, entities)
		require.NoError(t, err)

		path := filepath.Join(tmpDir, "out.yaml")
		b, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, []byte("test-bytes"), b)

		// second write should fail due to O_EXCL permissions
		// when writing files, so we don't override any existing files.
		err = Write(ctx, tmpDir, formatters, entities)
		require.Error(t, err)
		assert.ErrorContains(t, err, "file exists")
	})

	t.Run("creates intermediate directories", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir = t.TempDir()
			ctx    = context.Background()
		)

		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("ok")})

		rel := filepath.Join("nested", "dir", "file.yaml")
		entities := []FormattableEntity{{
			Content:      map[string]any{"x": 1},
			RelativePath: rel,
		}}

		err := Write(ctx, tmpDir, formatters, entities)
		require.NoError(t, err)

		_, statErr := os.Stat(filepath.Join(tmpDir, rel))
		require.NoError(t, statErr)
	})

	t.Run("formatter lookup by extension", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir = t.TempDir()
			ctx    = context.Background()
		)

		// Two formatters for two different extensions
		formatters := formatter.Setup(
			stubFormatter{exts: []string{"yaml"}, out: []byte("YAML-OUT")},
			stubFormatter{exts: []string{"txt"}, out: []byte("TXT-OUT")},
		)

		entities := []FormattableEntity{
			{Content: map[string]any{"a": 1}, RelativePath: "a.yaml"},
			{Content: map[string]any{"b": 2}, RelativePath: "b.txt"},
		}

		err := Write(ctx, tmpDir, formatters, entities)
		require.NoError(t, err)

		b1, e1 := os.ReadFile(filepath.Join(tmpDir, "a.yaml"))
		require.NoError(t, e1)
		assert.Equal(t, []byte("YAML-OUT"), b1)

		b2, e2 := os.ReadFile(filepath.Join(tmpDir, "b.txt"))
		require.NoError(t, e2)
		assert.Equal(t, []byte("TXT-OUT"), b2)
	})

	t.Run("unsupported extension error", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir = t.TempDir()
			ctx    = context.Background()
		)

		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("YAML-OUT")})
		entities := []FormattableEntity{
			{
				Content:      map[string]any{"a": 1},
				RelativePath: "object.json",
			},
		}

		err := Write(
			ctx,
			tmpDir,
			formatters,
			entities,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, formatter.ErrUnsupportedExtension)
	})

	t.Run("propagates formatter error", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir = t.TempDir()
			ctx    = context.Background()
		)

		formatterFail := errors.New("formatter failed")
		formatters := formatter.Setup(stubFormatter{
			exts: []string{"yaml"},
			err:  formatterFail,
		})

		entities := []FormattableEntity{
			{Content: map[string]any{"a": 1}, RelativePath: "out.yaml"},
		}

		err := Write(
			ctx,
			tmpDir,
			formatters,
			entities,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, formatterFail)
	})
}

func TestOverwriteFile(t *testing.T) {
	t.Run("creates new file when it doesn't exist", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("new-content")})

		entity := FormattableEntity{
			Content:      map[string]any{"key": "value"},
			RelativePath: filepath.Join(tmpDir, "new-file.yaml"),
		}

		err := OverwriteFile(formatters, entity)
		require.NoError(t, err)

		content, readErr := os.ReadFile(entity.RelativePath)
		require.NoError(t, readErr)
		assert.Equal(t, []byte("new-content"), content)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "existing-file.yaml")

		// Create existing file with original content
		err := os.WriteFile(filePath, []byte("original-content"), 0644)
		require.NoError(t, err)

		// Verify original content
		original, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, []byte("original-content"), original)

		// Overwrite with new content
		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("updated-content")})

		entity := FormattableEntity{
			Content:      map[string]any{"updated": "data"},
			RelativePath: filePath,
		}

		err = OverwriteFile(formatters, entity)
		require.NoError(t, err)

		// Verify file was overwritten
		updated, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, []byte("updated-content"), updated)
	})

	t.Run("propagates formatter error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		formatterErr := errors.New("formatting failed")
		formatters := formatter.Setup(stubFormatter{
			exts: []string{"yaml"},
			err:  formatterErr,
		})

		entity := FormattableEntity{
			Content:      map[string]any{"data": "value"},
			RelativePath: filepath.Join(tmpDir, "test.yaml"),
		}

		err := OverwriteFile(formatters, entity)
		require.Error(t, err)
		assert.ErrorIs(t, err, formatterErr)
		assert.ErrorContains(t, err, "formatting")
	})

	t.Run("returns error when writing to invalid directory", func(t *testing.T) {
		t.Parallel()

		formatters := formatter.Setup(stubFormatter{exts: []string{"yaml"}, out: []byte("content")})

		entity := FormattableEntity{
			Content:      map[string]any{"data": "value"},
			RelativePath: "/nonexistent/directory/that/does/not/exist/file.yaml",
		}

		err := OverwriteFile(formatters, entity)
		require.Error(t, err)
		assert.ErrorContains(t, err, "writing file")
	})
}
