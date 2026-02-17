package parser

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// pythonImportLexer defines tokens for Python import statement syntax.
// Keywords must appear before Ident so they are not consumed as identifiers.
var pythonImportLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "From", Pattern: `from`},
	{Name: "Import", Pattern: `import`},
	{Name: "As", Pattern: `as`},
	{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
	{Name: "Dot", Pattern: `\.`},
	{Name: "Star", Pattern: `\*`},
	{Name: "Comma", Pattern: `,`},
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "Whitespace", Pattern: `\s+`},
})

// importParser is built once at init time and is safe for concurrent use.
var importParser = participle.MustBuild[ImportStatement](
	participle.Lexer(pythonImportLexer),
	participle.Elide("Whitespace"),
)

// ImportStatement is the top-level grammar node. Every candidate line is either
// a "from X import ..." or a plain "import X, Y, Z".
type ImportStatement struct {
	From   *FromImport   `( @@`
	Simple *SimpleImport `| @@ )`
}

// FromImport represents: from [.]* [module] import (names | *)
//
// Dots is non-empty for relative imports (from . import x, from .. import y).
// Module is nil when importing directly from a package (from . import x).
type FromImport struct {
	Dots   []string    `"from" @Dot*`
	Module *ModulePath `@@?`
	Names  *ImportList `"import" @@`
}

// SimpleImport represents: import module1 [as alias], module2 ...
type SimpleImport struct {
	Modules []ModuleAlias `"import" @@ ("," @@)*`
}

// ModulePath captures a dotted name such as "mylib", "mylib.sub.deep".
type ModulePath struct {
	Parts []string `@Ident ("." @Ident)*`
}

// ModuleAlias is a module path with an optional alias: mylib [as ml].
type ModuleAlias struct {
	Path  ModulePath `@@`
	Alias *string    `("as" @Ident)?`
}

// ImportList is what follows "import" in a from-import: either "*" or a
// parenthesized/bare list of names, each optionally aliased.
type ImportList struct {
	Star  bool         `( @"*"`
	Names []ImportName `| "("? @@ ("," @@)* ","? ")"? )`
}

// ImportName is a single name inside a from-import list: name [as alias].
type ImportName struct {
	Name  string  `@Ident`
	Alias *string `("as" @Ident)?`
}
