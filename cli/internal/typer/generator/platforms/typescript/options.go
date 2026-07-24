package typescript

// TypeScriptOptions holds platform-specific options for TypeScript code generation.
// These can be passed via --option flags in the CLI, e.g.:
//
//	--option outputFileName=Events.ts
type TypeScriptOptions struct {
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated TypeScript file. Defaults to RudderTyper.ts"`
	// V1Compat additionally emits un-prefixed free functions bound to a default
	// client that lazily resolves window.rudderanalytics, as a drop-in for
	// consumers migrating from the npm rudder-typer v1 module. Off by default.
	V1Compat bool `mapstructure:"v1Compat" description:"Also emit un-prefixed free functions bound to window.rudderanalytics for drop-in compatibility with the npm rudder-typer v1 module. Defaults to false."`
}

// DefaultOptions returns the default TypeScript generation options.
func (g *Generator) DefaultOptions() any {
	return TypeScriptOptions{
		OutputFileName: "RudderTyper.ts",
	}
}
