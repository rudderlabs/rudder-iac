package resolver

// Resolver resolves a variable name to its string value.
// The two-return-value pattern lets callers distinguish "resolved to empty string"
// from "not found".
type Resolver interface {
	Resolve(name string) (value string, found bool)
}
