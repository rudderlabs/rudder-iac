package specdocs

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

// RenderDestinationMarkdown writes doc as Markdown. It reads only doc, so a
// generated docs bundle can render the same page from the same document.
func RenderDestinationMarkdown(w io.Writer, doc DestinationDoc) error {
	p := &printer{w: w}

	status := "verified"
	if !doc.Verified {
		status = "unverified, register it with " + envAssignments(doc.Enable)
	}
	p.linef("# %s (%s), definition_version %d, rudder-cli %s, %s", doc.Type, doc.DisplayName, doc.DefinitionVersion, doc.CLIVersion, status)
	p.line("")
	if doc.Source != nil {
		p.linef("Narrowed to `%s` sources, which connect as source token `%s`.", doc.Source.Type, doc.Source.Token)
		p.line("")
	}
	p.line("Each key goes under `spec.config`. `[]` marks the items of a list. A nested key marked required " +
		"is required once its parent block or list item is written, and a nested default applies only once " +
		"its parent block is written. A pattern applies to literal values; a key that takes a template " +
		"also accepts a `{{ path || fallback }}` template in its place.")
	p.line("")

	p.line("| key | type | required | default | allowed values or pattern | secret |")
	p.line("| --- | --- | --- | --- | --- | --- |")
	for _, f := range doc.Fields {
		p.row(code(f.Key), code(f.Type), requiredCell(f), defaultCell(f), allowedCell(f), secretCell(f))
	}

	p.line("")
	p.line("## Connection modes")
	p.line("")
	p.line("`connection_mode` needs an entry for each connected source token, set to one of the modes below. " +
		"A mode with required keys needs those keys in the config.")
	p.line("")
	p.line("| source token | modes | required keys |")
	p.line("| --- | --- | --- |")
	for _, token := range slices.Sorted(maps.Keys(doc.ConnectionModes)) {
		modes := doc.ConnectionModes[token]
		var required []string
		for _, mode := range slices.Sorted(maps.Keys(modes.RequiredKeys)) {
			required = append(required, fmt.Sprintf("%s: %s", mode, codeList(modes.RequiredKeys[mode], ", ")))
		}
		p.row(code(token), codeList(modes.Modes, ", "), strings.Join(required, "; "))
	}

	if len(doc.Patterns) > 0 {
		p.line("")
		p.line("## Patterns")
		p.line("")
		for _, name := range slices.Sorted(maps.Keys(doc.Patterns)) {
			pattern := doc.Patterns[name]
			text := fmt.Sprintf("- %s: %s. Matches %s", code(name), pattern.Message, code(pattern.Regex))
			if pattern.Reject != "" {
				text += ", and must not match " + code(pattern.Reject)
			}
			p.line(text + ".")
		}
	}

	if len(doc.CustomValidators) > 0 {
		p.line("")
		p.line("## Custom validators")
		p.line("")
		p.line("Go code also checks these rules, and this page cannot say what they require: " +
			codeList(doc.CustomValidators, ", ") + ". `rudder-cli validate` reports when one fails.")
	}

	if doc.SpecTemplate != "" {
		p.line("")
		p.line("## Spec template")
		p.line("")
		p.line("```yaml")
		p.write(doc.SpecTemplate)
		p.line("```")
	}
	return p.err
}

// RenderDestinationListMarkdown writes the listing as Markdown.
func RenderDestinationListMarkdown(w io.Writer, list DestinationList) error {
	p := &printer{w: w}
	p.linef("# Destination types, rudder-cli %s", list.CLIVersion)
	p.line("")
	p.line("Run `rudder-cli docs destination <type>` for the keys a type accepts.")
	p.line("")
	p.line("| type | display name | API type | definition versions | verified | source tokens |")
	p.line("| --- | --- | --- | --- | --- | --- |")

	var enable map[string]string
	for _, d := range list.Destinations {
		versions := make([]string, 0, len(d.DefinitionVersions))
		for _, v := range d.DefinitionVersions {
			versions = append(versions, fmt.Sprint(v))
		}
		verified := "yes"
		if !d.Verified {
			verified = "no"
			enable = d.Enable
		}
		p.row(code(d.Type), d.DisplayName, d.APIType, strings.Join(versions, ", "), verified, strings.Join(d.SourceTypes, ", "))
	}

	if enable != nil {
		p.line("")
		p.line("An unverified type registers only with " + envAssignments(enable) + ".")
	}
	return p.err
}

// RenderSourceListMarkdown writes the source listing as Markdown.
func RenderSourceListMarkdown(w io.Writer, list SourceList) error {
	p := &printer{w: w}
	p.linef("# Event stream source types, rudder-cli %s", list.CLIVersion)
	p.line("")
	p.line("A source spec's `type` takes one of these values. A destination accepts a source when it lists the " +
		"source's connection token, and `connection_mode` is keyed by that token.")
	p.line("")
	p.line("| type | connection token |")
	p.line("| --- | --- |")
	for _, s := range list.Sources {
		p.row(code(s.Type), code(s.Token))
	}
	return p.err
}

func requiredCell(f Field) string {
	var parts []string
	switch {
	case f.Required:
		parts = append(parts, "yes")
	case f.RequiredWhenParentPresent:
		parts = append(parts, "yes, once its parent is written")
	}
	if len(f.RequiredIf) > 0 {
		parts = append(parts, "if "+conditionsText(f.RequiredIf, " and "))
	}
	if len(f.RequiredUnless) > 0 {
		parts = append(parts, "unless "+conditionsText(f.RequiredUnless, " or "))
	}
	if len(f.RequiredWith) > 0 {
		parts = append(parts, "if "+codeList(f.RequiredWith, " or ")+" is set")
	}
	if len(f.ExcludedWith) > 0 {
		parts = append(parts, "not with "+codeList(f.ExcludedWith, " or "))
	}
	if len(f.ExcludedIf) > 0 {
		parts = append(parts, "not if "+conditionsText(f.ExcludedIf, " and "))
	}
	if len(f.CheckedBy) > 0 {
		parts = append(parts, "checked by "+codeList(f.CheckedBy, ", "))
	}
	return strings.Join(parts, "; ")
}

func conditionsText(conditions []Condition, joiner string) string {
	parts := make([]string, 0, len(conditions))
	for _, c := range conditions {
		parts = append(parts, fmt.Sprintf("%s is %s", code(c.Key), code(fmt.Sprint(c.Value))))
	}
	return strings.Join(parts, joiner)
}

func defaultCell(f Field) string {
	if f.Default == nil {
		return ""
	}
	text := code(fmt.Sprint(f.Default))
	if f.Default == "" {
		text = code(`""`)
	}
	if f.DefaultRequiresParent {
		text += " once its parent is written"
	}
	return text
}

func allowedCell(f Field) string {
	var parts []string
	if len(f.Enum) > 0 {
		values := make([]string, 0, len(f.Enum))
		for _, v := range f.Enum {
			values = append(values, fmt.Sprint(v))
		}
		text := codeList(values, ", ")
		if f.OpenEnum {
			text += ", or a template or `env.` value"
		}
		parts = append(parts, text)
	}
	if f.Pattern != "" {
		text := "pattern " + code(f.Pattern)
		if f.AcceptsTemplate {
			text += ", or a template"
		}
		parts = append(parts, text)
	}
	if f.Min != nil {
		parts = append(parts, fmt.Sprintf("at least %d", *f.Min))
	}
	if f.Max != nil {
		parts = append(parts, fmt.Sprintf("at most %d", *f.Max))
	}
	if len(f.OnlyForSourceTypes) > 0 {
		parts = append(parts, "only for "+codeList(f.OnlyForSourceTypes, ", "))
	}
	if f.Key == connectionModeKey {
		parts = append(parts, "see connection modes")
	}
	return strings.Join(parts, "; ")
}

func secretCell(f Field) string {
	if !f.Secret {
		return ""
	}
	if f.Var == "" {
		return "yes"
	}
	return "yes, " + code(f.Var)
}

func envAssignments(env map[string]string) string {
	parts := make([]string, 0, len(env))
	for _, name := range slices.Sorted(maps.Keys(env)) {
		parts = append(parts, code(name+"="+env[name]))
	}
	return strings.Join(parts, " and ")
}

func codeList(values []string, joiner string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, code(v))
	}
	return strings.Join(parts, joiner)
}

// code wraps s in a code span, using a fence longer than any backtick run in
// s, as CommonMark requires.
func code(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return fence + " " + s + " " + fence
	}
	return fence + s + fence
}

// printer keeps the first write error so the renderers read straight through.
type printer struct {
	w   io.Writer
	err error
}

func (p *printer) write(s string) {
	if p.err != nil {
		return
	}
	_, p.err = io.WriteString(p.w, s)
}

func (p *printer) line(s string) {
	p.write(s + "\n")
}

func (p *printer) linef(format string, args ...any) {
	p.line(fmt.Sprintf(format, args...))
}

// row writes a table row. A pipe inside a cell, code spans included, must be
// escaped or it ends the cell.
func (p *printer) row(cells ...string) {
	escaped := make([]string, len(cells))
	for i, cell := range cells {
		escaped[i] = strings.ReplaceAll(cell, "|", `\|`)
	}
	p.line("| " + strings.Join(escaped, " | ") + " |")
}
