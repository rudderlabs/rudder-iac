package kotlin

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/main.kt.tmpl
var kotlinTemplate string

//go:embed templates/typealias.tmpl
var typealiasTemplate string

func GenerateFile(path string, ctx *KotlinContext) (*core.File, error) {
	tmpl, err := template.New("kotlin").Parse(kotlinTemplate)
	if err != nil {
		return nil, err
	}

	// Parse and add sub-templates
	_, err = tmpl.New("typealias.tmpl").Parse(typealiasTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, ctx)
	if err != nil {
		return nil, err
	}

	return &core.File{
		Path:    path,
		Content: buf.String(),
	}, nil
}
