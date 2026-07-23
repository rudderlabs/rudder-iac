package export

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

var (
	secretStringType    = reflect.TypeOf(secret.String{})
	secretStringPtrType = reflect.TypeOf((*secret.String)(nil))
)

func (s *SpecExportData[Spec]) AttachSecretVariables(resourceType, defaultResourceID string, fields []handler.SecretField) error {
	if s == nil || s.Data == nil || len(fields) == 0 {
		return nil
	}

	fieldSet := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		fieldSet[field.JSONName] = struct{}{}
	}
	return attachSecretVariables(reflect.ValueOf(s.Data), resourceType, defaultResourceID, fieldSet)
}

func attachSecretVariables(v reflect.Value, resourceType, resourceID string, secretFields map[string]struct{}) error {
	value, ok := indirectValue(v)
	if !ok {
		return nil
	}

	switch value.Kind() {
	case reflect.Struct:
		if value.Type() == secretStringType {
			return nil
		}
		return attachSecretVariablesToStruct(value, resourceType, resourceID, secretFields)
	case reflect.Slice, reflect.Array:
		return attachSecretVariablesToSequence(value, resourceType, resourceID, secretFields)
	case reflect.Map:
		return attachSecretVariablesToMap(value, resourceType, resourceID, secretFields)
	}
	return nil
}

func indirectValue(v reflect.Value) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	return v, true
}

func attachSecretVariablesToSequence(v reflect.Value, resourceType, resourceID string, secretFields map[string]struct{}) error {
	for i := 0; i < v.Len(); i++ {
		if err := attachSecretVariables(v.Index(i), resourceType, resourceID, secretFields); err != nil {
			return err
		}
	}
	return nil
}

func attachSecretVariablesToStruct(v reflect.Value, resourceType, resourceID string, secretFields map[string]struct{}) error {
	t := v.Type()
	structResourceID := resourceIDForStruct(v, resourceID)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if err := attachSecretVariableToField(v.Field(i), field, resourceType, structResourceID, secretFields); err != nil {
			return err
		}
	}
	return nil
}

func attachSecretVariableToField(v reflect.Value, field reflect.StructField, resourceType, resourceID string, secretFields map[string]struct{}) error {
	jsonName := jsonFieldName(field)
	if _, ok := secretFields[jsonName]; ok && isSecretStringType(field.Type) {
		return setSecretVariableFromJSONName(field.Name, jsonName, v, resourceType, resourceID)
	}
	return attachSecretVariables(v, resourceType, resourceID, secretFields)
}

func setSecretVariableFromJSONName(fieldName, jsonName string, v reflect.Value, resourceType, resourceID string) error {
	if resourceID == "" {
		return fmt.Errorf("resource ID is required for secret field %s", jsonName)
	}
	return setSecretVariable(fieldName, v, secretVariableName(resourceType, resourceID, jsonName))
}

func attachSecretVariablesToMap(v reflect.Value, resourceType, resourceID string, secretFields map[string]struct{}) error {
	for _, key := range v.MapKeys() {
		if err := attachSecretVariablesToMapValue(v, key, resourceType, resourceID, secretFields); err != nil {
			return err
		}
	}
	return nil
}

func attachSecretVariablesToMapValue(v reflect.Value, key reflect.Value, resourceType, resourceID string, secretFields map[string]struct{}) error {
	value := v.MapIndex(key)
	if value.Kind() == reflect.Ptr {
		return attachSecretVariables(value, resourceType, resourceID, secretFields)
	}

	copyValue := reflect.New(value.Type()).Elem()
	copyValue.Set(value)
	if err := attachSecretVariables(copyValue, resourceType, resourceID, secretFields); err != nil {
		return err
	}
	v.SetMapIndex(key, copyValue)
	return nil
}

func resourceIDForStruct(v reflect.Value, fallback string) string {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() || jsonFieldName(field) != "id" {
			continue
		}
		if value, ok := stringFieldValue(v.Field(i)); ok {
			return value
		}
	}
	return fallback
}

func stringFieldValue(v reflect.Value) (string, bool) {
	value, ok := indirectValue(v)
	if !ok || value.Kind() != reflect.String || value.String() == "" {
		return "", false
	}
	return value.String(), true
}

func setSecretVariable(fieldName string, v reflect.Value, varName string) error {
	if !v.CanSet() {
		return fmt.Errorf("setting secret field %s: field is not settable", fieldName)
	}

	unknown := secret.NewUnknown(secret.WithVariableName(varName))
	switch v.Type() {
	case secretStringType:
		v.Set(reflect.ValueOf(unknown))
	case secretStringPtrType:
		v.Set(reflect.ValueOf(&unknown))
	}
	return nil
}

func secretVariableName(resourceType, resourceID, fieldName string) string {
	return sanitizeVariableName(resourceType + "_" + resourceID + "_" + fieldName)
}

func sanitizeVariableName(raw string) string {
	var b strings.Builder
	previousWasLowerOrDigit := false
	for _, r := range raw {
		previousWasLowerOrDigit = writeVariableRune(&b, r, previousWasLowerOrDigit)
	}
	return ensureVariableNameStart(b.String())
}

func writeVariableRune(b *strings.Builder, r rune, previousWasLowerOrDigit bool) bool {
	switch {
	case r >= 'A' && r <= 'Z':
		if previousWasLowerOrDigit {
			b.WriteByte('_')
		}
		b.WriteRune(r)
		return false
	case r >= 'a' && r <= 'z':
		b.WriteRune(r - 'a' + 'A')
		return true
	case r >= '0' && r <= '9':
		b.WriteRune(r)
		return true
	default:
		b.WriteByte('_')
		return false
	}
}

func ensureVariableNameStart(name string) string {
	if name == "" {
		return "_"
	}
	if name[0] >= '0' && name[0] <= '9' {
		return "_" + name
	}
	return name
}

func isSecretStringType(t reflect.Type) bool {
	return t == secretStringType || t == secretStringPtrType
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	if name != "" {
		return name
	}
	return field.Name
}
