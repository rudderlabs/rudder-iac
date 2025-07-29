package kotlin

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/main.kt.tmpl
var kotlinTemplate string

func GenerateFile(path string, t string, ctx *RootContext) (*core.File, error) {
	tmpl, err := template.New("kotlin").Parse(t)
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
