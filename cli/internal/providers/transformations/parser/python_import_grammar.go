package parser

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// pythonImportLexer uses a single Ident rule for all identifiers, including keywords.
//
// Keywords (from, import, as) are matched as literal values in the grammar rather than
// as separate token types. This prevents the lexer from greedily splitting identifiers
// like "importlib" into Import("import") + Ident("lib"), which would break parsing of
// "import importlib".
var pythonImportLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
	{Name: "Dot", Pattern: `\.`},
	{Name: "Star", Pattern: `\*`},
	{Name: "Comma", Pattern: `,`},
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "Whitespace", Pattern: `\s+`},
})

// importParser is built once and is safe for concurrent use.
//
// UseLookahead(MaxLookahead) enables full backtracking, which is needed for
// FromImport: the optional @@? (ModulePath) may greedily consume "import" as an
// identifier before the following "import" literal fails, requiring a backtrack.
// Example: "from . import util" — ModulePath grabs "import", then "import" literal
// fails, so participle must backtrack to Module=nil and re-match "import" correctly.
var importParser = participle.MustBuild[ImportStatement](
	participle.Lexer(pythonImportLexer),
	participle.Elide("Whitespace"),
	participle.UseLookahead(participle.MaxLookahead),
)

// ImportStatement is the top-level grammar node. Every candidate line is either
// a "from X import ..." or a plain "import X, Y, Z".
type ImportStatement struct {
	From   *FromImport   `( @@`
	Simple *SimpleImport `| @@ )`
}

// FromImport represents: from [.]* [module] import (names | *)
//
// Dots captures leading dots for relative imports (e.g. from . or from ..).
// Module is nil for bare relative imports like "from . import x".
//
// The grammar uses two alternatives to distinguish "from . import x" (no module)
// from "from .pkg import x" or "from mylib import x" (with module path).
// This avoids the ambiguity where @@? would consume "import" as a module name
// before the following "import" literal fails.
type FromImport struct {
	Dots   []string    `"from" @Dot*`
	Module *ModulePath `( @@ "import" | "import" )`
	Names  *ImportList `@@`
}

// SimpleImport represents: import module1 [as alias], module2 ...
type SimpleImport struct {
	Modules []ModuleAlias `"import" @@ ("," @@)*`
}

// ModulePath captures a dotted name such as "mylib" or "mylib.sub.deep".
type ModulePath struct {
	Parts []string `@Ident ("." @Ident)*`
}

// ModuleAlias is a module path with an optional alias: mylib [as ml].
type ModuleAlias struct {
	Path  ModulePath `@@`
	Alias *string    `("as" @Ident)?`
}

// ImportList is what follows "import" in a from-import: either "*" or a
// parenthesized or bare list of names, each optionally aliased.
type ImportList struct {
	Star  bool         `( @"*"`
	Names []ImportName `| "("? @@ ("," @@)* ","? ")"? )`
}

// ImportName is a single name inside a from-import list: name [as alias].
type ImportName struct {
	Name  string  `@Ident`
	Alias *string `("as" @Ident)?`
}
