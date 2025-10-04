package kotlin

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/Main.kt.tmpl
var kotlinTemplate string

//go:embed templates/typealias.tmpl
var typealiasTemplate string

//go:embed templates/dataclass.tmpl
var dataclassTemplate string

//go:embed templates/rudderanalytics.tmpl
var rudderanalyticsTemplate string

//go:embed templates/enum.tmpl
var enumTemplate string

func GenerateFile(path string, ctx *KotlinContext) (*core.File, error) {
	funcMap := template.FuncMap{
		"indent": func(level int) string {
			return strings.Repeat("    ", level)
		},
	}

	tmpl, err := template.New("kotlin").Funcs(funcMap).Parse(kotlinTemplate)
	if err != nil {
		return nil, err
	}

	// Parse and add sub-templates
	_, err = tmpl.New("typealias.tmpl").Parse(typealiasTemplate)
	if err != nil {
		return nil, err
	}

	_, err = tmpl.New("dataclass.tmpl").Parse(dataclassTemplate)
	if err != nil {
		return nil, err
	}

	_, err = tmpl.New("rudderanalytics.tmpl").Parse(rudderanalyticsTemplate)
	if err != nil {
		return nil, err
	}

	_, err = tmpl.New("enum.tmpl").Parse(enumTemplate)
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
