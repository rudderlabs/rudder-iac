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
	var tmpl *template.Template

	funcMap := template.FuncMap{
		"indent": func(level int, text string) string {
			indentStr := "    "
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = indentStr + line
				}
			}
			return strings.Join(lines, "\n")
		},
		"include": func(name string, data interface{}) (string, error) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, name, data)
			return buf.String(), err
		},
		"add": func(a, b int) int {
			return a + b
		},
		"mkSlice": func(args ...any) []any {
			return args
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
