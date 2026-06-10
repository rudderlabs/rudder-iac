package secret

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

var (
	stringType     = reflect.TypeOf(String{})
	importableType = reflect.TypeOf(ImportableSecret{})
)

// StringDecodeHook converts a plain string into a String or ImportableSecret
// (or pointers to them) during a mapstructure decode. Spec maps carry secrets
// as bare strings (after YAML load and variable substitution); without this
// hook mapstructure cannot populate a secret field, since the struct has no
// exported value field to map onto.
func StringDecodeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}

		// reflect, not a type assertion, so named string types convert too.
		raw := reflect.ValueOf(data).String()

		switch to {
		case stringType:
			return New(raw), nil
		case reflect.PointerTo(stringType):
			s := New(raw)
			return &s, nil
		case importableType:
			return ImportableSecret{String: New(raw)}, nil
		case reflect.PointerTo(importableType):
			s := ImportableSecret{String: New(raw)}
			return &s, nil
		}
		return data, nil
	}
}
