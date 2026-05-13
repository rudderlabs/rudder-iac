package typescript

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/RudderTyper.ts.tmpl
var typescriptTemplate string

//go:embed templates/disclaimer.tmpl
var disclaimerTemplate string

//go:embed templates/interface.tmpl
var interfaceTemplate string

//go:embed templates/ruddertyper.tmpl
var ruddertyperTemplate string

func GenerateFile(path string, ctx *TSContext) (*core.File, error) {
	funcMap := template.FuncMap{
		"escapeString":  EscapeTSStringLiteral,
		"escapeComment": EscapeJSDocComment,
		"formatLiteral": FormatTSLiteral,
	}

	tmpl, err := template.New("typescript").Funcs(funcMap).Parse(typescriptTemplate)
	if err != nil {
		return nil, err
	}

	for name, src := range map[string]string{
		"disclaimer.tmpl":  disclaimerTemplate,
		"interface.tmpl":   interfaceTemplate,
		"ruddertyper.tmpl": ruddertyperTemplate,
	} {
		if _, err = tmpl.New(name).Parse(src); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, ctx); err != nil {
		return nil, err
	}

	return &core.File{
		Path:    path,
		Content: buf.String(),
	}, nil
}
