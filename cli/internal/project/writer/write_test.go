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
