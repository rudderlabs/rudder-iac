package golang

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"strconv"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// render lays out ctx and formats it with go/format, so a template that
// produces invalid Go fails generation instead of shipping.
func render(ctx *GoContext) (string, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"quote":   strconv.Quote,
		"comment": comment,
		"inline":  inlineComment,
	}).ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing templates: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ruddertyper.go.tmpl", ctx); err != nil {
		return "", fmt.Errorf("executing templates: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("formatting generated code: %w", err)
	}
	return string(src), nil
}
