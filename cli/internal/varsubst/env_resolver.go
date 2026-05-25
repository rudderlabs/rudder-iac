package varsubst

import (
	"os"
	"strings"
)

const defaultEnvPrefix = "RUDDER_"

type EnvResolver struct {
	vars   map[string]string
	prefix string
}

func NewEnvResolver(prefix string) *EnvResolver {
	if prefix == "" {
		prefix = defaultEnvPrefix
	}

	varResolver := &EnvResolver{
		vars:   make(map[string]string),
		prefix: prefix,
	}

	for _, envEntry := range os.Environ() {
		envKey, envValue, hasValue := strings.Cut(envEntry, "=")
		if !hasValue {
			continue
		}
		if !strings.HasPrefix(envKey, varResolver.prefix) {
			continue
		}

		resolvedName := strings.TrimPrefix(envKey, varResolver.prefix)
		varResolver.vars[resolvedName] = envValue
	}

	return varResolver
}

func (r *EnvResolver) Resolve(name string) (string, bool) {
	if r == nil {
		return "", false
	}

	value, found := r.vars[name]
	return value, found
}
