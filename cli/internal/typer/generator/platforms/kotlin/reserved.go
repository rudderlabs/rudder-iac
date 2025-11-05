package kotlin

// Source: https://kotlinlang.org/docs/keyword-reference.html

var KotlinHardKeywords = map[string]bool{
	// Hard keywords
	"as":        true,
	"break":     true,
	"class":     true,
	"continue":  true,
	"do":        true,
	"else":      true,
	"false":     true,
	"for":       true,
	"fun":       true,
	"if":        true,
	"in":        true,
	"interface": true,
	"is":        true,
	"null":      true,
	"object":    true,
	"package":   true,
	"return":    true,
	"super":     true,
	"this":      true,
	"throw":     true,
	"true":      true,
	"try":       true,
	"typealias": true,
	"typeof":    true,
	"val":       true,
	"var":       true,
	"when":      true,
	"while":     true,
}

// aditional keywords are kept here for completeness, but they are not strictly handled
// as they do not require special treatment in properties.

var KotlinSoftKeywords = map[string]bool{
	"by":          true,
	"catch":       true,
	"constructor": true,
	"delegate":    true,
	"dynamic":     true,
	"field":       true,
	"file":        true,
	"finally":     true,
	"get":         true,
	"import":      true,
	"init":        true,
	"param":       true,
	"property":    true,
	"receiver":    true,
	"set":         true,
	"setparam":    true,
	"value":       true,
	"where":       true,
}

var KotlinOtherKeywords = map[string]bool{
	// Modifier keywords
	"abstract":    true,
	"actual":      true,
	"annotation":  true,
	"companion":   true,
	"const":       true,
	"crossinline": true,
	"data":        true,
	"enum":        true,
	"expect":      true,
	"external":    true,
	"final":       true,
	"infix":       true,
	"inline":      true,
	"inner":       true,
	"internal":    true,
	"lateinit":    true,
	"noinline":    true,
	"open":        true,
	"operator":    true,
	"out":         true,
	"override":    true,
	"private":     true,
	"protected":   true,
	"public":      true,
	"reified":     true,
	"sealed":      true,
	"suspend":     true,
	"tailrec":     true,
	"vararg":      true,

	// Special identifiers
	"it": true,
}
