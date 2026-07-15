package typescript

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

//go:embed templates/RudderTyper.ts.tmpl
var typescriptTemplate string

//go:embed templates/disclaimer.tmpl
var disclaimerTemplate string

//go:embed templates/interface.tmpl
var interfaceTemplate string

//go:embed templates/typealias.tmpl
var typealiasTemplate string

//go:embed templates/ruddertyper.tmpl
var ruddertyperTemplate string

// compatFreeFunctionName returns the un-prefixed free-function name for the v1
// compatibility layer. Track methods are named `track<Event>` on the class but
// were un-prefixed free functions in the v1 module (e.g. `sourceCreated`). We strip
// the `track` prefix off the already-registered method name (rather than re-deriving
// from the event) so the NameRegistry's collision suffixes are preserved — two
// events that camelCase to the same name stay distinct (`fooBar`, `fooBar1`).
// identify/group/page keep their names.
//
// Residual collision risk: a track event literally named "Identify"/"Page"/"Group"
// strips to a name that clashes with the singleton method. Not guarded here yet —
// see the PR discussion.
func compatFreeFunctionName(m TSAnalyticsMethod) string {
	if m.SDKMethodName == "track" && strings.HasPrefix(m.Name, "track") {
		stripped := strings.TrimPrefix(m.Name, "track")
		if stripped == "" {
			return m.Name
		}
		r := []rune(stripped)
		r[0] = unicode.ToLower(r[0])
		return string(r)
	}
	return m.Name
}

func GenerateFile(path string, ctx *TSContext) (*core.File, error) {
	funcMap := template.FuncMap{
		"escapeString":  EscapeTSStringLiteral,
		"escapeComment": EscapeJSDocComment,
		"formatLiteral": FormatTSLiteral,
		"compatName":    compatFreeFunctionName,
	}

	tmpl, err := template.New("typescript").Funcs(funcMap).Parse(typescriptTemplate)
	if err != nil {
		return nil, err
	}

	for name, src := range map[string]string{
		"disclaimer.tmpl":  disclaimerTemplate,
		"interface.tmpl":   interfaceTemplate,
		"typealias.tmpl":   typealiasTemplate,
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
