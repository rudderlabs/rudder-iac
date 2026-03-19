package swift

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/RudderTyper.swift.tmpl
var swiftTemplate string

//go:embed templates/disclaimer.tmpl
var disclaimerTemplate string

//go:embed templates/typealias.tmpl
var typealiasTemplate string

//go:embed templates/enum.tmpl
var enumTemplate string

//go:embed templates/multitype.tmpl
var multitypeTemplate string

//go:embed templates/struct.tmpl
var structTemplate string

//go:embed templates/variant.tmpl
var variantTemplate string

//go:embed templates/ruddertyperanalytics.tmpl
var ruddertyperanalyticsTemplate string

func GenerateFile(path string, ctx *SwiftContext) (*core.File, error) {
	var tmpl *template.Template

	funcMap := template.FuncMap{
		"escapeString":  EscapeSwiftStringLiteral,
		"escapeComment": EscapeSwiftComment,
		"formatLiteral": FormatSwiftLiteral,
		"indent": func(level int, text string) string {
			pad := strings.Repeat("    ", level)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = pad + line
				}
			}
			return strings.Join(lines, "\n")
		},
		// include allows templates to call other named templates and capture output as a string.
		// This is needed for recursive struct rendering (nested structs inside variant enums).
		"include": func(name string, data any) (string, error) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, name, data)
			return buf.String(), err
		},
		// mkSlice packs variadic args into []any so templates can pass multiple values
		// as a single argument (e.g. passing both indent level and struct to structwithindent).
		"mkSlice": func(args ...any) []any {
			return args
		},
	}

	var err error
	tmpl, err = template.New("swift").Funcs(funcMap).Parse(swiftTemplate)
	if err != nil {
		return nil, err
	}

	for name, src := range map[string]string{
		"disclaimer.tmpl":           disclaimerTemplate,
		"typealias.tmpl":            typealiasTemplate,
		"enum.tmpl":                 enumTemplate,
		"multitype.tmpl":            multitypeTemplate,
		"variant.tmpl":              variantTemplate,
		"ruddertyperanalytics.tmpl": ruddertyperanalyticsTemplate,
	} {
		if _, err = tmpl.New(name).Parse(src); err != nil {
			return nil, err
		}
	}

	// struct.tmpl defines "structwithindent" internally — parsed separately
	// so the named template is registered and callable from variant.tmpl and the root.
	if _, err = tmpl.New("struct.tmpl").Parse(structTemplate); err != nil {
		return nil, err
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
