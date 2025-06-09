package importer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

func generateFromTemplate(tmplContent []byte, data any) ([]byte, error) {
	tmpl, err := template.New("property").Funcs(template.FuncMap{
		"toYAML": func(v interface{}) string {
			b, err := yaml.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		},
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}).Parse(string(tmplContent))

	if err != nil {
		return nil, fmt.Errorf("parsing property template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing property template: %w", err)
	}

	return buf.Bytes(), nil
}
