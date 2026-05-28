package resolver

import (
	"os"
	"strings"
)

const defaultEnvPrefix = "RUDDER_"

type envResolver struct {
	vars map[string]string
}

func NewEnvResolver() Resolver {
	return newEnvResolverFromEnviron(os.Environ())
}

func newEnvResolverFromEnviron(environ []string) Resolver {
	vars := make(map[string]string)
	for _, env := range environ {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, defaultEnvPrefix) {
			// Last-write-wins: if the same RUDDER_-prefixed key appears more than
			// once in environ, the later occurrence silently overrides the earlier
			// one. This mirrors the OS behaviour on lookup.
			vars[strings.TrimPrefix(key, defaultEnvPrefix)] = value
		}
	}

	return &envResolver{vars: vars}
}

func (r *envResolver) Resolve(name string) (string, bool) {
	value, found := r.vars[name]
	return value, found
}
