package typescript

// Sources:
//   - https://www.typescriptlang.org/docs/handbook/2/keyword-types.html
//   - https://github.com/microsoft/TypeScript/blob/main/src/compiler/types.ts
//
// tsReservedWords includes JavaScript reserved words, future-reserved words, and
// TypeScript contextual keywords that cannot be used as identifiers without
// renaming (we do not emit the rare quoted-identifier form). Pre-defined types
// such as `string`, `number`, `boolean`, `null`, `undefined`, `any`, `unknown`,
// `void`, and `never` are also blocked because using them as identifiers
// shadows the built-in type and produces confusing diagnostics under
// `strict: true`.
var tsReservedWords = map[string]bool{
	// Reserved keywords
	"break":      true,
	"case":       true,
	"catch":      true,
	"class":      true,
	"const":      true,
	"continue":   true,
	"debugger":   true,
	"default":    true,
	"delete":     true,
	"do":         true,
	"else":       true,
	"enum":       true,
	"export":     true,
	"extends":    true,
	"false":      true,
	"finally":    true,
	"for":        true,
	"function":   true,
	"if":         true,
	"import":     true,
	"in":         true,
	"instanceof": true,
	"new":        true,
	"null":       true,
	"return":     true,
	"super":      true,
	"switch":     true,
	"this":       true,
	"throw":      true,
	"true":       true,
	"try":        true,
	"typeof":     true,
	"var":        true,
	"void":       true,
	"while":      true,
	"with":       true,

	// Strict-mode reserved
	"as":         true,
	"implements": true,
	"interface":  true,
	"let":        true,
	"package":    true,
	"private":    true,
	"protected":  true,
	"public":     true,
	"static":     true,
	"yield":      true,

	// Contextual / TypeScript-specific
	"any":         true,
	"async":       true,
	"await":       true,
	"boolean":     true,
	"constructor": true,
	"declare":     true,
	"from":        true,
	"get":         true,
	"is":          true,
	"keyof":       true,
	"module":      true,
	"namespace":   true,
	"never":       true,
	"number":      true,
	"of":          true,
	"readonly":    true,
	"require":     true,
	"set":         true,
	"string":      true,
	"symbol":      true,
	"type":        true,
	"undefined":   true,
	"unknown":     true,
}
