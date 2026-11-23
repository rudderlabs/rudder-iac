package typescript

// TypeScriptOptions holds platform-specific options for TypeScript code generation.
// These can be passed via --option flags in the CLI, e.g.:
//
//	--option outputFileName=Events.ts
type TypeScriptOptions struct {
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated TypeScript file. Defaults to RudderTyper.ts"`
}

// DefaultOptions returns the default TypeScript generation options.
func (g *Generator) DefaultOptions() any {
	return TypeScriptOptions{
		OutputFileName: "RudderTyper.ts",
	}
}
