package converter

import (
	"fmt"
	"reflect"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConfigProperty defines how a property in an API config object maps to local YAML config and vice versa.
type ConfigProperty struct {
	ToLocalFunc   ToLocalFunc
	FromLocalFunc FromLocalFunc
	// LocalKey is the gjson dot path of this property in local config
	// (e.g. "webhook_url"). Empty for derived properties that have no
	// single local key, such as Discriminator.
	LocalKey string
	// SourceTypes, when set via Gated, restricts LocalKey to destinations
	// connected to one of these local source types. Empty means the key is
	// allowed for every connected source type.
	SourceTypes []string
}

// Gated restricts prop's local key to the given local source types. The
// property must carry a LocalKey (i.e. not be a Discriminator); the registry
// rejects gated properties without one at registration.
//
// We added this abstraction of gating because in future in validation layer of explicit connections,
// we would need to identify the config keys which can only be added if a corresponding sourcetype
// is connected to it.
func Gated(prop ConfigProperty, sourceTypes ...string) ConfigProperty {
	prop.SourceTypes = sourceTypes
	return prop
}

// FromLocalFunc modifies an API config JSON object using local config information.
type FromLocalFunc func(config, local string) (string, error)

// ToLocalFunc modifies a local config JSON object by extracting data from an API config JSON object.
type ToLocalFunc func(local, config string) (string, error)

// Simple returns a ConfigProperty that maps an API config key to a local config key and vice versa.
func Simple(apiKey, localKey string, filters ...ValueFilter) ConfigProperty {
	return ConfigProperty{
		LocalKey:      localKey,
		FromLocalFunc: copyFromLocal(apiKey, localKey, filters...),
		ToLocalFunc:   copyToLocal(apiKey, localKey),
	}
}

// bucketName -> bucket_name ( Simple )
// abcdef -> abc-> {def} ( ArrayWith ) ( )
// abcghi -> abc-> {ghi}

// skipWhilelistEvents > skip_list_events->white_listed
// skipBlacklistEvents > skip_list_events->black_listed

type ValueFilter func(a any) bool

type APINestedObject struct {
	LocalKey  string
	NestedKey string
}

type localNestedObject struct {
	APIKey    string
	NestedKey string
}

// SkipZeroValue returns true if the value is Go's zero value or an empty slice.
func SkipZeroValue(a any) bool {
	switch v := a.(type) {
	case []any:
		return len(v) == 0
	default:
		return reflect.ValueOf(a).IsZero()
	}
}

// Conditional returns a ConfigProperty that maps an API config key to a local config key
// only if the provided condition is satisfied for that API config.
func Conditional(apiKey, localKey string, condition ConfigConditionFunc) ConfigProperty {
	return ConfigProperty{
		LocalKey:      localKey,
		FromLocalFunc: copyFromLocal(apiKey, localKey),
		ToLocalFunc:   copyToLocalConditional(apiKey, localKey, condition),
	}
}

// ConfigConditionFunc checks an API config object for a condition.
type ConfigConditionFunc func(config string) bool

// Equals returns a ConfigConditionFunc that is true if the API config contains
// the specified key and it has the specified value.
func Equals(key, value string) ConfigConditionFunc {
	return func(config string) bool {
		r := gjson.Get(config, key)
		return r.Exists() && r.Value() == value
	}
}

// Discriminator returns a ConfigProperty that is not stored directly in local config.
// The corresponding API config value is set based on the provided DiscriminatorValues.
func Discriminator(apiKey string, values DiscriminatorValues) ConfigProperty {
	return ConfigProperty{
		FromLocalFunc: discriminatorValue(apiKey, values),
		ToLocalFunc:   func(local, config string) (string, error) { return local, nil },
	}
}

// DiscriminatorValues maps local config keys to API discriminator values.
type DiscriminatorValues map[string]any

func ArrayWithStrings(rootAPIKey, nestedAPIField, localKey string) ConfigProperty {
	return ConfigProperty{
		LocalKey: localKey,
		FromLocalFunc: func(config, local string) (string, error) {
			result := config
			v := gjson.Get(local, localKey)
			if v.Exists() && v.Value() != nil {
				switch a := v.Value().(type) {
				case []any:
					contents := []any{}
					for _, i := range a {
						contents = append(contents, map[string]any{nestedAPIField: i})
					}

					if len(contents) > 0 {
						r, err := sjson.Set(result, rootAPIKey, contents)
						if err != nil {
							return result, err
						}
						result = r
					}
				default:
					return result, fmt.Errorf("provided value was not an array")
				}
			}
			return result, nil
		},
		ToLocalFunc: func(local, config string) (string, error) {
			result := local

			r := gjson.Get(config, rootAPIKey)
			if r.Exists() && r.IsArray() {
				contents := []any{}
				for _, i := range r.Value().([]any) {
					if m, ok := i.(map[string]any); ok {
						if v, ok := m[nestedAPIField]; ok {
							contents = append(contents, v)
						}
					}
				}
				s, err := sjson.Set(result, localKey, contents)
				if err != nil {
					return result, err
				}
				result = s
			}

			return result, nil
		},
	}
}

func GetInverseFields(fields map[string]any) map[string]any {
	inverseFields := map[string]any{}
	for a, t := range fields {
		switch fieldVal := t.(type) {
		case string:
			inverseFields[fieldVal] = a
		case APINestedObject:
			localKey := fieldVal.LocalKey
			nestedKey := fieldVal.NestedKey
			inverseFields[localKey] = localNestedObject{APIKey: a, NestedKey: nestedKey}
		}
	}
	return inverseFields
}

func ArrayWithObjects(rootAPIKey, localKey string, fields map[string]any) ConfigProperty {
	inverseFields := GetInverseFields(fields)

	return ConfigProperty{
		LocalKey: localKey,
		FromLocalFunc: func(config, local string) (string, error) {
			result := config
			v := gjson.Get(local, localKey)
			if v.Exists() && v.Value() != nil {
				switch a := v.Value().(type) {
				case []any:
					contents := GetAPIValue(a, inverseFields)
					if len(contents) >= 0 {
						r, err := sjson.Set(result, rootAPIKey, contents)
						if err != nil {
							return result, err
						}
						result = r
					}
				default:
					return result, fmt.Errorf("provided value was not an array")
				}
			}
			return result, nil
		},
		ToLocalFunc: func(local, config string) (string, error) {
			result := local

			r := gjson.Get(config, rootAPIKey)
			if r.Exists() && r.IsArray() {
				contents := GetLocalValue(r.Value().([]any), fields)
				s, err := sjson.Set(result, localKey, contents)
				if err != nil {
					return result, err
				}
				result = s
			}

			return result, nil
		},
	}
}

func GetLocalValue(configValue []any, fields map[string]any) []any {
	contents := []any{}
	for _, i := range configValue {
		tv := map[string]any{}

		if av, ok := i.(map[string]any); ok {
			for af, v := range av {
				if tf, ok := fields[af]; ok {
					switch fieldVal := tf.(type) {
					case string:
						tv[fieldVal] = v
					case APINestedObject:
						tfValues := []any{}
						localKey := fieldVal.LocalKey
						nestedKey := fieldVal.NestedKey

						for _, nestedObj := range v.([]any) {
							tfValues = append(tfValues, nestedObj.(map[string]any)[nestedKey])
						}
						tv[localKey] = tfValues
					}
				}
			}
		}

		if len(tv) > 0 {
			contents = append(contents, tv)
		}
	}
	return contents
}

func GetAPIValue(localValue []any, fields map[string]any) []any {
	contents := []any{}
	for _, i := range localValue {
		av := map[string]any{}

		if tv, ok := i.(map[string]any); ok {
			for localField, localFieldValue := range tv {
				switch fieldVal := fields[localField].(type) {
				case string:
					av[fieldVal] = localFieldValue
				case localNestedObject:
					tfValues := []any{}
					apiKey := fieldVal.APIKey
					nestedKey := fieldVal.NestedKey

					for _, nestedVal := range localFieldValue.([]any) {
						nestedObj := map[string]any{
							nestedKey: nestedVal,
						}
						tfValues = append(tfValues, nestedObj)
					}
					av[apiKey] = tfValues
				}
			}
		}

		if len(av) > 0 {
			contents = append(contents, av)
		}
	}
	return contents
}

func applyFilters(a any, filters []ValueFilter) bool {
	for _, o := range filters {
		if o(a) {
			return false
		}
	}

	return true
}

func copyFromLocal(apiKey, localKey string, options ...ValueFilter) FromLocalFunc {
	return func(config, local string) (string, error) {
		result := config
		v := gjson.Get(local, localKey)
		if v.Exists() && v.Value() != nil && applyFilters(v.Value(), options) {
			sresult, err := sjson.Set(result, apiKey, v.Value())
			if err != nil {
				return result, err
			}
			result = sresult
		}

		return result, nil
	}
}

func copyToLocal(apiKey, localKey string) ToLocalFunc {
	return func(local, config string) (string, error) {
		r := gjson.Get(config, apiKey)
		if r.Exists() {
			s, err := sjson.Set(local, localKey, r.Value())
			if err != nil {
				return local, err
			}
			local = s
		}

		return local, nil
	}
}

func copyToLocalConditional(apiKey, localKey string, condition ConfigConditionFunc) ToLocalFunc {
	return func(local, config string) (string, error) {
		if !condition(config) {
			return local, nil
		}

		return copyToLocal(apiKey, localKey)(local, config)
	}
}

func discriminatorValue(apiKey string, values DiscriminatorValues) FromLocalFunc {
	return func(config, local string) (string, error) {
		for k, v := range values {
			r := gjson.Get(local, k)

			if !r.Exists() {
				continue
			}

			switch r.Type {
			case gjson.JSON:
				if r.IsArray() && len(r.Value().([]any)) == 0 {
					continue
				}
			case gjson.String:
				if r.Value() == "" {
					continue
				}
			}

			return sjson.Set(config, apiKey, v)
		}

		return config, nil
	}
}
