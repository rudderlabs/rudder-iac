package secret

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

var stringType = reflect.TypeOf(String{})

// StringDecodeHook converts a plain string into a String (or *String) during a
// mapstructure decode. Spec maps carry secrets as bare strings (after YAML load
// and variable substitution); without this hook mapstructure cannot populate a
// String field, since the struct has no exported fields to map onto.
func StringDecodeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}

		// reflect, not a type assertion, so named string types convert too.
		switch to {
		case stringType:
			return New(reflect.ValueOf(data).String()), nil
		case reflect.PointerTo(stringType):
			s := New(reflect.ValueOf(data).String())
			return &s, nil
		}
		return data, nil
	}
}
