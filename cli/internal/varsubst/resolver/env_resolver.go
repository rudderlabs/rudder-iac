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
			vars[strings.TrimPrefix(key, defaultEnvPrefix)] = value
		}
	}

	return &envResolver{vars: vars}
}

func (r *envResolver) Resolve(name string) (string, bool) {
	value, found := r.vars[name]
	return value, found
}
