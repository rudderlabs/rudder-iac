package state

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func DereferenceByReflection(v any, state *State) error {
	return dereferenceByReflectionValue(reflect.ValueOf(v), state)
}

func dereferenceByReflectionValue(val reflect.Value, state *State) error {
	if !val.IsValid() {
		return nil
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}

		// Special handling for *PropertyRef
		if val.Type() == reflect.TypeOf(&resources.PropertyRef{}) {
			propRef := val.Interface().(*resources.PropertyRef)
			if err := resolvePropertyRef(propRef, state); err != nil {
				return err
			}
			return nil
		}

		// For other pointers, dereference and continue
		return dereferenceByReflectionValue(val.Elem(), state)

	case reflect.Struct:
		// Special handling for PropertyRef by value
		if val.Type() == reflect.TypeOf(resources.PropertyRef{}) {
			if err := resolvePropertyRef(val.Addr().Interface().(*resources.PropertyRef), state); err != nil {
				return err
			}

			return nil
		}

		// Recursively process all struct fields
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)

			// Skip unexported fields
			if !field.CanInterface() {
				continue
			}

			if err := dereferenceByReflectionValue(field, state); err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		// Recursively process all slice/array elements
		for i := 0; i < val.Len(); i++ {
			if err := dereferenceByReflectionValue(val.Index(i), state); err != nil {
				return err
			}
		}

	case reflect.Map:
		// Recursively process all map values
		for _, key := range val.MapKeys() {
			mapValue := val.MapIndex(key)
			if err := dereferenceByReflectionValue(mapValue, state); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolvePropertyRef(propRef *resources.PropertyRef, state *State) error {
	if propRef == nil {
		return nil
	}

	if propRef.Resolve != nil {
		resource := state.GetResource(propRef.URN)
		if resource == nil {
			return fmt.Errorf("referred resource '%s' does not exist", propRef.URN)
		}

		value, err := propRef.Resolve(resource.OutputRaw)
		if err != nil {
			return fmt.Errorf("resolving property ref for %s: %w", propRef.URN, err)
		}
		propRef.IsResolved = true
		propRef.Value = value
		return nil
	}

	resource := state.GetResource(propRef.URN)
	if resource == nil {
		return fmt.Errorf("referred resource '%s' does not exist", propRef.URN)
	}

	resourceData := resource.Data()
	if resourceData == nil {
		return nil
	}

	value, exists := resourceData[propRef.Property]
	if !exists {
		return fmt.Errorf("property '%s' does not exist in resource '%s'", propRef.Property, propRef.URN)
	}

	// Convert value to string for ResolvedValue
	var stringValue string
	switch v := value.(type) {
	case string:
		stringValue = v
	case fmt.Stringer:
		stringValue = v.String()
	default:
		stringValue = fmt.Sprintf("%v", v)
	}

	// Populate the IsResolved & Value fields in place
	propRef.IsResolved = true
	propRef.Value = stringValue

	return nil
}
