package resources

import (
	"reflect"
)

type PropertyRef struct {
	URN        string                                `json:"urn"`
	Property   string                                `json:"property"`
	IsResolved bool                                  `json:"resolved"`
	Value      string                                `json:"value,omitempty"`
	Resolve    func(stateOutput any) (string, error) `json:"-"`
}

func collectReferences(v any) []*PropertyRef {
	if v == nil {
		return nil
	}

	var refs []*PropertyRef

	switch v := v.(type) {
	case []map[string]any:
		for _, vv := range v {
			refs = append(refs, collectReferences(vv)...)
		}
	case map[string]any:
		for _, vv := range v {
			refs = append(refs, collectReferences(vv)...)
		}
	case []any:
		for _, vv := range v {
			refs = append(refs, collectReferences(vv)...)
		}
	case *PropertyRef:
		// a *PropertyRef can be nil, for example when the categoryId is not set
		// in that case we don't want to add it to refs
		if v != nil {
			refs = append(refs, v)
		}
	case PropertyRef:
		refs = append(refs, &v)
	case ResourceData:
		for _, vv := range v {
			refs = append(refs, collectReferences(vv)...)
		}
	}

	return refs
}

func collectReferencesByReflection(v any) []*PropertyRef {
	if v == nil {
		return nil
	}

	var refs []*PropertyRef

	// Use reflection to inspect the value of v
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// If v is a slice or array, iterate over its elements
		for i := 0; i < val.Len(); i++ {
			refs = append(refs, collectReferencesByReflection(val.Index(i).Interface())...)
		}
		return refs
	case reflect.Map:
		// If v is a map, iterate over its values
		for _, key := range val.MapKeys() {
			refs = append(refs, collectReferencesByReflection(val.MapIndex(key).Interface())...)
		}
		return refs
	case reflect.Struct:
		// Recursively collect references from the fields of the struct
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			refs = append(refs, collectReferencesByReflection(field.Interface())...)
		}
		return refs
	}

	// For all other types (including pointers), delegate to collectReferences
	// which handles *PropertyRef and other specific types
	return collectReferences(v)
}
